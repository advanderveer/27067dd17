package bcast

import (
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestCertificateGeneration(t *testing.T) {
	cert, _, err := createCertificates(nil)
	test.Ok(t, err)
	block, _ := pem.Decode(cert)

	cert2, err := x509.ParseCertificate(block.Bytes)
	test.Ok(t, err) //should be parsable
	test.Equals(t, true, cert2.IsCA)
	test.Equals(t, 1, len(cert2.IPAddresses))
}

func TestCertificateGenerationWitExternalIP(t *testing.T) {
	cert, _, err := createCertificates(net.ParseIP("192.168.1.1"))
	test.Ok(t, err)

	block, _ := pem.Decode(cert)
	cert2, err := x509.ParseCertificate(block.Bytes)
	test.Ok(t, err) //should be parsable
	test.Equals(t, 2, len(cert2.IPAddresses))
	test.Equals(t, "192.168.1.1", cert2.IPAddresses[1].String())
}
