package testsec

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/security"

	. "github.com/onsi/gomega"
)

func TestSecurityManager(t *testing.T) {
	g := NewWithT(t)

	ca, err := generateCA()
	g.Expect(err).To(BeNil())

	exchangeTimeout := 500 * time.Millisecond
	mgr := security.NewManagerWithCertObtainer(newTestCertificateObtainerWithCA(testSpiffeID, &ca, exchangeTimeout))

	var prevCrt *tls.Certificate
	var prevCa *x509.CertPool

	for i := 0; i < 5; i++ {
		crt := mgr.GetCertificate()
		bundle := mgr.GetCABundle()

		g.Expect(crt).ToNot(Equal(prevCrt))
		if prevCa != nil {
			g.Expect(bundle).To(Equal(prevCa))
		}

		g.Expect(verify(crt, bundle)).To(BeNil())
		prevCrt, prevCa = crt, bundle

		<-time.After(2 * exchangeTimeout)
	}
}
