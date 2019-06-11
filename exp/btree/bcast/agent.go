package bcast

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// Agent take care of the broadcast that initiates connections
type Agent struct {
	h     *Handler
	ca    []byte
	ln    net.Listener
	peers *Peers
	svr   *http.Server
}

// NewAgent initiates a new broadcast client
func NewAgent(bind string, peers *Peers) (c *Agent, err error) {
	c = &Agent{peers: peers}

	c.ln, err = net.Listen("tcp", bind)
	if err != nil {
		return nil, errors.Wrap(err, "failed to listen")
	}

	var certf string
	var keyf string
	c.ca, certf, keyf, err = GenerateCertification()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate server certification")
	}

	c.h = NewHandler()
	c.svr = &http.Server{Handler: c.h}

	go func() {
		err = c.svr.ServeTLS(c.ln, certf, keyf)
		if err != nil && err != http.ErrServerClosed {
			panic(err) //@TODO handle errors more gracefully
		}
	}()

	//@TODO maintain a bunch of active clients to which this agent also
	//writes messages

	return
}

// Addr returns the host and port the server is listening on
func (a *Agent) Addr() (ip net.IP, port uint16, ca []byte) {
	host, ps, _ := net.SplitHostPort(a.ln.Addr().String())
	pi, _ := strconv.ParseUint(ps, 10, 16)
	return net.ParseIP(host), uint16(pi), a.ca
}

// Join the network with an initial other host
func (a *Agent) Join(host net.IP, port uint16, ca []byte) (err error) {
	err = a.peers.Put(host.String(), fmt.Sprintf(`{"port": %d, "ca": "%s"}`, port, base64.StdEncoding.EncodeToString(ca)))
	return
}

// Write a message to this agents connected peers, if any
func (a *Agent) Write(msg *Msg) (err error) {
	//@TODO also push to peers we connected to
	return a.h.Write(msg)
}

// Read the next message received by peers, if any
func (a *Agent) Read() (msg *Msg, err error) { return a.h.Read() }

// Close the agent by stopping the server
func (a *Agent) Close() (err error) {
	err = a.h.Close()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	return a.svr.Shutdown(ctx)
}
