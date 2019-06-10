package security

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
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

type certs struct {
	cert []byte
	key  []byte
	ca   []byte
}

type certificateManager struct {
	sync.RWMutex
	certs *certs
}

func (m *certificateManager) ClientCredentials() (credentials.TransportCredentials, error) {
	cert, err := tls.X509KeyPair(m.certs.cert, m.certs.key)
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(m.certs.ca); !ok {
		return nil, errors.New("failed to append ca cert to pool")
	}

	return credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
	}), nil
}

func (m *certificateManager) ServerCredentials() (credentials.TransportCredentials, error) {
	cert, err := tls.X509KeyPair(m.certs.cert, m.certs.key)
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	if ok := caPool.AppendCertsFromPEM(m.certs.ca); !ok {
		return nil, errors.New("failed to append ca cert to pool")
	}

	return credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caPool,
	}), nil
}

func (m *certificateManager) readCertificates() error {
	var err error
	newCerts := &certs{}

	if newCerts.cert, err = ioutil.ReadFile(certFile); err != nil {
		return err
	}

	if newCerts.key, err = ioutil.ReadFile(keyFile); err != nil {
		return err
	}

	if newCerts.ca, err = ioutil.ReadFile(caFile); err != nil {
		return err
	}

	m.setCertificates(newCerts)
	logrus.Info("Certificates successfully read")
	return nil
}

func (m *certificateManager) getCertificates() *certs {
	m.RLock()
	defer m.RUnlock()
	return m.certs
}

func (m *certificateManager) setCertificates(c *certs) {
	m.Lock()
	defer m.Unlock()
	m.certs = c
}

func NewCertificateManager() CertificateManager {
	cm := &certificateManager{}
	if err := cm.readCertificates(); err != nil {
		logrus.Error(err)
	}
	return cm
}
