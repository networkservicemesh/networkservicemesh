package testsec

import (
	"crypto/x509"
	"testing"

	. "github.com/onsi/gomega"
)

func TestSimpleCertCreation(t *testing.T) {
	g := NewWithT(t)

	ca, err := generateCA()
	g.Expect(err).To(BeNil())

	caX509, err := x509.ParseCertificate(ca.Certificate[0])
	g.Expect(err).To(BeNil())

	roots := x509.NewCertPool()
	roots.AddCert(caX509)

	crt, err := generateKeyPair(spiffeID1, testDomain, &ca)
	g.Expect(err).To(BeNil())

	err = verify(&crt, roots)
	g.Expect(err).To(BeNil())
}
