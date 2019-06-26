package test

import (
	"crypto/tls"
	"github.com/networkservicemesh/networkservicemesh/security"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	frequency = 3 * time.Second
)

type testExchangeCertificateObtainer struct {
	caTLS     *tls.Certificate
	frequency time.Duration
	errorCh   chan error
	spiffeID  string
}

func (t *testExchangeCertificateObtainer) ObtainCertificates() <-chan *security.RetrievedCerts {
	certCh := make(chan *security.RetrievedCerts, 1)

	bundle, err := caToBundle(t.caTLS)
	if err != nil {
		logrus.Error(err)
		t.errorCh <- err
		close(certCh)
		return certCh
	}

	go func() {
		defer close(certCh)

		for {
			logrus.Info("Generating new x509 certificate...")
			cert, err := generateKeyPair(t.spiffeID, t.caTLS)
			if err != nil {
				logrus.Error(err)
				t.errorCh <- err
				return
			}

			certCh <- &security.RetrievedCerts{
				CABundle: bundle,
				TLSCert:  &cert,
			}
			<-time.After(t.frequency)
		}
	}()

	return certCh
}

func (*testExchangeCertificateObtainer) Stop() {
}

func (*testExchangeCertificateObtainer) Error() error {
	return nil
}

//
//func newExchangeCertObtainer(frequency time.Duration) (security.CertificateObtainer, error) {
//	ca, err := generateCA()
//	if err != nil {
//		return nil, err
//	}
//
//	return newExchangeCertObtainerWithCA(&ca, frequency)
//}

func newExchangeCertObtainerWithCA(spiffeID string, caTLS *tls.Certificate, frequency time.Duration) (security.CertificateObtainer, error) {
	return &testExchangeCertificateObtainer{
		frequency: frequency,
		caTLS:     caTLS,
		spiffeID:  spiffeID,
	}, nil
}
