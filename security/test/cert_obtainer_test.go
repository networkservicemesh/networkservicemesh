package testsec

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestCertificateObtainer(t *testing.T) {
	RegisterTestingT(t)

	ca, err := generateCA()
	Expect(err).To(BeNil())

	obt := newTestCertificateObtainerWithCA(testSpiffeID, &ca, 500*time.Millisecond)
	certCh := obt.CertificateCh()

	for i := 0; i < 5; i++ {
		c := <-certCh

		Expect(verify(c.TLSCert, c.CABundle)).To(BeNil())
	}
}
