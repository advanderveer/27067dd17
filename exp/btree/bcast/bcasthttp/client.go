package bcasthttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	bcast "github.com/advanderveer/27067dd17/exp/btree/bcast"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
)

var (
	// ErrClientClosed is returned when a client has encountered the failed state
	// or shut down gracefully.
	ErrClientClosed = errors.New("client closed and can no longer be used")
)

// Client is broadcast client that interacts with one broadcast server
type Client struct {
	ip   net.IP
	port int

	// http components
	to   time.Duration
	tr   *http.Transport
	c    *http.Client
	logs *log.Logger

	// push components
	push *http.Request
	pw   io.WriteCloser
	enc  *gob.Encoder
}

// NewClient initiates a client with a server that is expected to run at the
// provided ip and port. It opens up the push stream right away
func NewClient(logw io.Writer, ip net.IP, port int, ca *x509.Certificate, to time.Duration) (c *Client) {
	roots := x509.NewCertPool() //start with empty pool
	if ca != nil {
		roots.AddCert(ca)
	}

	c = &Client{
		to:   to,
		ip:   ip,
		port: port,
		logs: log.New(logw, fmt.Sprintf("bcast/client[%s:%d] ", ip, port), 0),
	}

	c.tr = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs: roots,
		},
	}

	http2.ConfigureTransport(c.tr)
	c.c = &http.Client{
		Transport: c.tr,
	}

	c.push, c.pw = c.startPush()
	c.enc = gob.NewEncoder(c.pw)
	return
}

func (c *Client) startPush() (req *http.Request, pw io.WriteCloser) {
	loc := fmt.Sprintf("https://%s:%d/push", c.ip, c.port)
	pr, pw := io.Pipe()
	req, _ = http.NewRequest(http.MethodPut, loc, pr)

	// run the request in another thread. If the function ever ends it will
	// mean the client is no longer usable and needs to be re-created.
	go func() {
		defer pw.Close()

		// if there is a timeout, specificy it on the request
		ctx := context.Background()
		var cancel func()
		if c.to > 0 {
			ctx, cancel = context.WithTimeout(ctx, c.to)
			defer cancel()
		}

		// execute the request
		resp, err := c.c.Do(req.WithContext(ctx))
		if err != nil {
			c.logs.Printf("[DEBUG] push request failed with error: %v", err)
			return
		}

		if resp.StatusCode != http.StatusOK {
			c.logs.Printf("[DEBUG] server returned push response with unexpected status: %s", resp.Status)
		}
	}()

	return req, pw
}

// Write a message to the server
func (c *Client) Write(msg *bcast.Msg) (err error) {
	err = c.enc.Encode(msg)
	if err == io.ErrClosedPipe {
		return ErrClientClosed
	} else if err != nil {
		return errors.Wrap(err, "failed to encode and push message")
	}

	return nil
}

// Close will shutdown any open requests
func (c *Client) Close() (err error) {
	err = c.pw.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close push writer")
	}

	c.tr.CloseIdleConnections()
	return nil
}
