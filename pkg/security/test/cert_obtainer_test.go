package testsec

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestCertificateObtainer(t *testing.T) {
	g := NewWithT(t)

	ca, err := generateCA()
	g.Expect(err).To(BeNil())

	obt := newTestCertificateObtainerWithCA(spiffeID1, &ca, 500*time.Millisecond)
	certCh := obt.CertificateCh()

	for i := 0; i < 5; i++ {
		c := <-certCh

		g.Expect(verify(c.TLSCert, c.CABundle)).To(BeNil())
	}
}
