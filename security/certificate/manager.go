package security

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"
	"sync"
)

const (
	certFile = "/etc/certs/cert.pem"
	keyFile  = "/etc/certs/key.pem"
	caFile   = "/etc/certs/ca.pem"
)

type CertificateManager interface {
	ClientCredentials() (credentials.TransportCredentials, error)
	ServerCredentials() (credentials.TransportCredentials, error)
}

type CertificateObtainer interface {
	ObtainCertificates() <-chan *certs
	Stop()
	Error() error
}

type certs struct {
	cert []byte
	key  []byte
	ca   []byte
}

type certificateManager struct {
	sync.RWMutex
	certs   *certs
	readyCh chan struct{}
}

func (m *certificateManager) ClientCredentials() (credentials.TransportCredentials, error) {
	c := m.getCertificates()
	cert, err := tls.X509KeyPair(c.cert, c.key)
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(c.ca); !ok {
		return nil, errors.New("failed to append ca cert to pool")
	}

	return credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
	}), nil
}

func (m *certificateManager) ServerCredentials() (credentials.TransportCredentials, error) {
	c := m.getCertificates()
	cert, err := tls.X509KeyPair(c.cert, c.key)
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(c.ca); !ok {
		return nil, errors.New("failed to append ca cert to pool")
	}

	return credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
	}), nil
}

func (m *certificateManager) getCertificates() *certs {
	logrus.Infof("Waiting for certificates...")
	<-m.readyCh
	logrus.Infof("Certificates were obtained")

	m.RLock()
	defer m.RUnlock()
	return m.certs
}

func (m *certificateManager) setCertificates(c *certs) {
	m.Lock()
	defer m.Unlock()

	if m.certs == nil {
		close(m.readyCh)
	}
	m.certs = c
}

func (m *certificateManager) exchangeCertificates(obtainer CertificateObtainer) {
	logrus.Infof("exchangeCertificates %v", obtainer)
	certCh := obtainer.ObtainCertificates()
	logrus.Infof("ObtainCertificates() = %v", certCh)

	for {
		c, ok := <-certCh
		if ok {
			logrus.Info("ok")
			m.setCertificates(c)
			continue
		}
		logrus.Info("ne ok")
		if err := obtainer.Error(); err != nil {
			logrus.Errorf(err.Error())
		}
		return
	}
}

func NewCertificateManager(obtainer CertificateObtainer) CertificateManager {
	cm := &certificateManager{
		readyCh: make(chan struct{}),
	}
	go cm.exchangeCertificates(obtainer)
	return cm
}
