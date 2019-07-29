package testsec

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/security"

	. "github.com/onsi/gomega"
)

func TestSecurityManager(t *testing.T) {
	RegisterTestingT(t)

	ca, err := generateCA()
	Expect(err).To(BeNil())

	exchangeTimeout := 500 * time.Millisecond
	mgr := security.NewManagerWithCertObtainer(newTestCertificateObtainerWithCA(testSpiffeID, &ca, exchangeTimeout))

	var prevCrt *tls.Certificate
	var prevCa *x509.CertPool

	for i := 0; i < 5; i++ {
		crt := mgr.GetCertificate()
		bundle := mgr.GetCABundle()

		Expect(crt).ToNot(Equal(prevCrt))
		if prevCa != nil {
			Expect(bundle).To(Equal(prevCa))
		}

		verify(crt, bundle)
		prevCrt, prevCa = crt, bundle

		<-time.After(2 * exchangeTimeout)
	}
}
