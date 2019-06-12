package bcasthttp

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
)

// ParseCertificate parses a pem encoded cert
func ParseCertificate(pemd []byte) (c *x509.Certificate, err error) {
	block, _ := pem.Decode(pemd)
	c, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse provided CA certificate")
	}

	return
}

// createCertificates will setup a simple self-signed tls certificate and key that
// is its own ca that can be pinned by clients.
func createCertificates(eip net.IP) (cert, key []byte, err error) {

	//random serial nr for the certicate
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %s", err)
	}

	// create a certificate that will be made publicly available for connecting
	// peers that is its own ca. Clients are expected to add it instead of the
	// normal root cas.
	template := x509.Certificate{
		IsCA:                  true,
		SerialNumber:          serial,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
	}

	// if provided with an external ip this server is reachable on, add it as well
	if eip != nil {
		template.IPAddresses = append(template.IPAddresses, eip)
	}

	// generate an eliptic curve key pair for the certificate
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key pair for certificate: %v", err)
	}

	// create the complete certificate as bytes
	c, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	// encode as pem data in-memory
	cbuf := bytes.NewBuffer(nil)
	if err := pem.Encode(cbuf, &pem.Block{Type: "CERTIFICATE", Bytes: c}); err != nil {
		return nil, nil, fmt.Errorf("failed to encode certificate: %v", err)
	}

	// serialze private key
	kb, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %v", err)
	}

	// and encode as pem also
	kbuf := bytes.NewBuffer(nil)
	if err := pem.Encode(kbuf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}); err != nil {
		return nil, nil, fmt.Errorf("failed to encode private key: %v", err)
	}

	return cbuf.Bytes(), kbuf.Bytes(), nil
}
