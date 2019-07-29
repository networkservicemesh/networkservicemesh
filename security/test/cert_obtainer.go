package testsec

import (
	"crypto/tls"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/security"
)

type testCertificateObtainer struct {
	errCh  chan error
	certCh <-chan *security.Response
}

func newTestCertificateObtainerWithCA(spiffeID string, ca *tls.Certificate, frequency time.Duration) security.CertificateObtainer {
	errCh := make(chan error)
	certCh := newCertCh(spiffeID, ca, frequency, errCh)

	return &testCertificateObtainer{
		errCh:  errCh,
		certCh: certCh,
	}
}

func (o *testCertificateObtainer) Stop() {
}

func (o *testCertificateObtainer) ErrorCh() <-chan error {
	return o.errCh
}

func (o *testCertificateObtainer) CertificateCh() <-chan *security.Response {
	return o.certCh
}

func newCertCh(spiffeID string, ca *tls.Certificate, frequency time.Duration, errCh chan<- error) <-chan *security.Response {
	certCh := make(chan *security.Response, 1)

	bundle, err := caToBundle(ca)
	if err != nil {
		errCh <- err
		close(certCh)
		return certCh
	}

	go func() {
		defer close(certCh)

		for {
			logrus.Info("Generating new x509 certificate...")
			cert, err := generateKeyPair(spiffeID, ca)
			if err != nil {
				errCh <- err
				return
			}

			certCh <- &security.Response{
				CABundle: bundle,
				TLSCert:  &cert,
			}
			<-time.After(frequency)
		}
	}()

	return certCh
}
