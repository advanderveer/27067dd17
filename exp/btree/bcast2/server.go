package bcast

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Server serves broadcast clients
type Server struct {
	logs *log.Logger
	ln   *net.TCPListener
	srv  *http.Server
	eip  net.IP
	cert []byte
	key  []byte
}

// NewServer will initialize the server
func NewServer(logw io.Writer, ip net.IP, port int, ipd AddrDetector, h http.Handler) (s *Server, err error) {
	s = &Server{logs: log.New(logw, "bcast/server: ", 0)}

	s.srv = &http.Server{
		ErrorLog: s.logs,
		Handler:  h,

		// Configuration below is taken as-is (without further checks) from:
		// https://blog.cloudflare.com/exposing-go-on-the-internet/
		TLSConfig: &tls.Config{

			// Causes servers to use Go's default ciphersuite preferences,
			// which are tuned to avoid attacks. Does nothing on clients.
			PreferServerCipherSuites: true,

			// Only use curves which have assembly implementations
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519, // Go 1.8 only
			},

			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,

				// Best disabled, as they don't provide Forward Secrecy,
				// but might be necessary for some clients
				// tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				// tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			},

			// we will use one self-signed certificate
			Certificates: make([]tls.Certificate, 1),
		},

		// ReadTimeout covers the time from when the connection is accepted to
		// when the request body is fully read. What this means for streaming
		// endpoints is discussed here: https://github.com/golang/go/issues/16100
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	s.eip = ipd.DetectExternalIP(true)
	s.cert, s.key, err = createCertificates(s.eip)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create server certificates")
	}

	s.srv.TLSConfig.Certificates[0], err = tls.X509KeyPair(s.cert, s.key)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse use certificates")
	}

	addr := &net.TCPAddr{IP: ip, Port: port}
	s.ln, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to listen on '%s'", addr)
	}

	return
}

// Serve will start serving broadcast requests
func (s *Server) Serve() (err error) {
	err = s.srv.ServeTLS(s.ln, "", "")
	if err != nil && err != http.ErrServerClosed {
		return errors.Wrap(err, "failed to serve")
	}

	return nil
}

// Info returns descriptive info about the server that would
// allow others to connect to it.
func (s *Server) Info() (iip, eip net.IP, port int, ca []byte) {
	addr := s.ln.Addr().(*net.TCPAddr)
	return addr.IP, s.eip, addr.Port, s.cert
}

// Close the server
func (s *Server) Close(ctx context.Context) (err error) {
	err = s.srv.Shutdown(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to shutdown gracefully")
	}

	err = s.ln.Close()
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		return errors.Wrap(err, "failed to close listener")
	}

	return nil
}
