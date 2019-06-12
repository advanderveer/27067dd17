package bcasthttp

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/gob"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	bcast "github.com/advanderveer/27067dd17/exp/btree/bcast"
	"github.com/advanderveer/go-test"
	"golang.org/x/net/http2"
)

var _ bcast.Reader = &Handler{}

func TestBareHandler(t *testing.T) {
	s := httptest.NewServer(NewHandler(os.Stderr, 0))
	defer s.Close()

	c := s.Client()
	defer c.CloseIdleConnections()

	// without tls and http2 it will return bad request
	resp, err := c.Get(s.URL)
	test.Ok(t, err)
	test.Equals(t, http.StatusBadRequest, resp.StatusCode)

	t.Run("with tls and http2", func(t *testing.T) {
		h := NewHandler(os.Stderr, 0)
		s := httptest.NewUnstartedServer(h)

		s.TLS = &tls.Config{NextProtos: []string{"h2"}}

		s.StartTLS()
		cert, err := x509.ParseCertificate(s.TLS.Certificates[0].Certificate[0])
		test.Ok(t, err)

		roots := x509.NewCertPool()
		roots.AddCert(cert)

		c = &http.Client{Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: roots,
			},
		}}

		resp, err := c.Get(s.URL)
		test.Ok(t, err)
		test.Equals(t, http.StatusNotFound, resp.StatusCode)

		t.Run("push garbage to server", func(t *testing.T) {
			resp, err := c.Post(s.URL+"/push", "application/octet-stream", bytes.NewReader([]byte{0x01}))
			test.Ok(t, err)
			test.Equals(t, http.StatusUnsupportedMediaType, resp.StatusCode)
		})

		t.Run("push non garbage after close", func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			enc := gob.NewEncoder(buf)
			test.Ok(t, enc.Encode(&bcast.Msg{Foo: "bar"}))

			test.Ok(t, h.Close()) //close handler

			// should now return unvailable to new connections
			resp, _ = c.Post(s.URL+"/push", "application/octet-stream", ioutil.NopCloser(buf))
			test.Equals(t, http.StatusServiceUnavailable, resp.StatusCode)

			//close server, breaking down connections
			s.Close()

			//error should now indicate it is closed
			_, err := h.Read()
			test.Equals(t, io.EOF, err)

			//post should not cause handler to panic, just show refused error
			_, err = c.Post(s.URL+"/push", "application/octet-stream", ioutil.NopCloser(buf))
			test.Assert(t, err != nil, "expected some error")
		})
	})
}
