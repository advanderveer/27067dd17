package bcast

import (
	"crypto/x509"
	"io/ioutil"
	"testing"

	"github.com/advanderveer/go-test"
)

func TestCertificateGeneration(t *testing.T) {
	cert, certf, keyf, err := GenerateCertification()
	test.Ok(t, err)

	_, err = ioutil.ReadFile(certf)
	test.Ok(t, err)

	_, err = ioutil.ReadFile(keyf)
	test.Ok(t, err)

	cert2, err := x509.ParseCertificate(cert)
	test.Ok(t, err) //should be parsable
	test.Equals(t, true, cert2.IsCA)
}
