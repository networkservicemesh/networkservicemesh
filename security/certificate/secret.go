package security

import (
	"github.com/sirupsen/logrus"
	"io/ioutil"
)

type secretCertObtainer struct {
	certFile string
	keyFile  string
	caFile   string
	errorCh  chan error
	stopCh   chan<- struct{}
}

func NewSecretCertObtainer(certFile, keyFile, caFile string) CertificateObtainer {
	return &secretCertObtainer{
		certFile: certFile,
		keyFile:  keyFile,
		caFile:   caFile,
		errorCh:  make(chan error, 1),
		stopCh:   make(chan struct{}),
	}
}

func (s *secretCertObtainer) ObtainCertificates() <-chan *certs {
	certCh := make(chan *certs)
	defer close(certCh)

	c, err := s.readCertificates()
	if err != nil {
		s.errorCh <- err
	} else {
		certCh <- c
	}

	return certCh
}

func (s *secretCertObtainer) Stop() {
	close(s.stopCh)
}

func (s *secretCertObtainer) Error() error {
	return <-s.errorCh
}

func (s *secretCertObtainer) readCertificates() (*certs, error) {
	var err error
	newCerts := &certs{}

	if newCerts.cert, err = ioutil.ReadFile(s.certFile); err != nil {
		return nil, err
	}

	if newCerts.key, err = ioutil.ReadFile(s.keyFile); err != nil {
		return nil, err
	}

	if newCerts.ca, err = ioutil.ReadFile(s.caFile); err != nil {
		return nil, err
	}

	logrus.Info("Certificates successfully read")
	return newCerts, nil
}
