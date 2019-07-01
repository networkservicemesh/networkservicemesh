package testsec

import (
	"crypto/tls"
	"github.com/networkservicemesh/networkservicemesh/security"
)

const (
	testDomain   = "test.com"
	testSpiffeID = "spiffe://test.com/test"
)

type testSimpleCertificateObtainer struct {
	cert *security.RetrievedCerts
}

func (t *testSimpleCertificateObtainer) ObtainCertificates() <-chan *security.RetrievedCerts {
	certCh := make(chan *security.RetrievedCerts, 1)
	certCh <- t.cert
	close(certCh)
	return certCh
}

func (*testSimpleCertificateObtainer) Stop() {
}

func (*testSimpleCertificateObtainer) Error() error {
	return nil
}

func newSimpleCertObtainer(spiffeID string) (security.CertificateObtainer, error) {
	ca, err := generateCA()
	if err != nil {
		return nil, err
	}

	return newSimpleCertObtainerWithCA(spiffeID, &ca)
}

func newSimpleCertObtainerWithCA(spiffeID string, caTLS *tls.Certificate) (security.CertificateObtainer, error) {
	caPool, err := caToBundle(caTLS)
	if err != nil {
		return nil, err
	}

	crt, err := generateKeyPair(spiffeID, caTLS)
	if err != nil {
		return nil, err
	}

	return &testSimpleCertificateObtainer{
		cert: &security.RetrievedCerts{
			CABundle: caPool,
			TLSCert:  &crt,
		},
	}, nil
}
