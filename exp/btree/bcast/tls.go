package bcast

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"time"
)

// GenerateCertification will setup a simple self-signed tls certificate that is
// its own ca that can be pinned by clients.
func GenerateCertification() (cert []byte, certp, keyp string, err error) {

	//random serial nr for the certicate
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
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

	// generate an eliptic curve key pair for the certificate
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate key pair for certificate: %v", err)
	}

	// create the complete certificate as bytes
	cert, err = x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create certificate: %v", err)
	}

	// write the certificate to a temporary file
	certf, err := ioutil.TempFile("", "bchain_cert_")
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create temp cert file: %v", err)
	}

	defer certf.Close()
	if err := pem.Encode(certf, &pem.Block{Type: "CERTIFICATE", Bytes: cert}); err != nil {
		return nil, "", "", fmt.Errorf("failed to encode certificate: %v", err)
	}

	// write the private key
	keyf, err := ioutil.TempFile("", "tls_key_")
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create temporary file for certificate key: %v", err)
	}

	defer keyf.Close()
	kb, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to marshal private key: %v", err)
	}

	if err := pem.Encode(keyf, &pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}); err != nil {
		return nil, "", "", fmt.Errorf("failed to encode private key: %v", err)
	}

	return cert, certf.Name(), keyf.Name(), nil
}
