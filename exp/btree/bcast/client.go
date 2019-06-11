package bcast

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/pkg/errors"
	"golang.org/x/net/http2"
)

// Client is used for communication to a single peer
type Client struct {
	ip   net.IP
	port uint16
	cl   *http.Client
	wc   io.WriteCloser
	enc  *gob.Encoder
	rc   io.ReadCloser
	dec  *gob.Decoder
}

// NewClient sets up a client for peer communication
func NewClient(ip net.IP, port uint16, cab []byte) (c *Client, err error) {
	ca, err := x509.ParseCertificate(cab)
	if err != nil {
		return nil, fmt.Errorf("failed to parse peer's CA certificate: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(ca)
	c = &Client{ip: ip, port: port, cl: &http.Client{Transport: &http2.Transport{
		TLSClientConfig: &tls.Config{RootCAs: roots},
	}}}

	c.wc, err = c.push()
	if err != nil {
		return nil, fmt.Errorf("failed to setup push comms: %v", err)
	}

	c.enc = gob.NewEncoder(c.wc)

	c.rc, err = c.pull()
	if err != nil {
		return nil, fmt.Errorf("failed to setup pull comms: %v", err)
	}

	c.dec = gob.NewDecoder(c.rc)
	return c, nil
}

func (c *Client) Write(msg *Msg) (err error) {
	return c.enc.Encode(msg)
}

func (c *Client) Read() (msg *Msg, err error) {
	msg = new(Msg)
	return msg, c.dec.Decode(msg)
}

func (c *Client) push() (pw io.WriteCloser, r error) {
	pr, pw := io.Pipe()
	req, err := http.NewRequest(http.MethodPut, fmt.Sprintf("https://%s:%d/push", c.ip, c.port), ioutil.NopCloser(pr))
	if err != nil {
		panic("bcast/client: " + err.Error())
	}

	// In a push we do not expect the response to be returned until
	// the stream ends.
	go func() {
		_, err = c.cl.Do(req)
		if err != nil {
			panic(err) //@TODO how to handle async errors?
		}
	}()

	return pw, nil
}

func (c *Client) pull() (pr io.ReadCloser, err error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s:%d/pull", c.ip, c.port), nil)
	if err != nil {
		panic("bcast/client: " + err.Error())
	}

	resp, err := c.cl.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to perform pull request")
	}

	pr, pw := io.Pipe()
	go func() {
		_, err := io.Copy(pw, resp.Body)
		if err != nil {
			panic(err) //@TODO handle copy errors
		}

	}()

	return pr, nil
}

// Close the open communcation channels with the peer
func (c *Client) Close() (err error) {
	err = c.wc.Close()
	if err != nil {
		return err
	}

	err = c.rc.Close()
	if err != nil {
		return err
	}

	return
}
