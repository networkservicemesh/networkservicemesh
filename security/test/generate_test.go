package testsec

import (
	"crypto/x509"
	"testing"

	. "github.com/onsi/gomega"
)

func TestSimpleCertCreation(t *testing.T) {
	RegisterTestingT(t)

	ca, err := generateCA()
	Expect(err).To(BeNil())

	caX509, err := x509.ParseCertificate(ca.Certificate[0])
	Expect(err).To(BeNil())

	roots := x509.NewCertPool()
	roots.AddCert(caX509)

	crt, err := generateKeyPair(testSpiffeID, &ca)
	Expect(err).To(BeNil())

	err = verify(&crt, roots)
	Expect(err).To(BeNil())
}
