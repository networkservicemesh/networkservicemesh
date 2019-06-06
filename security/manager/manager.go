package security

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/security/manager/apis"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"net"
	"sync"
)

const (
	certFile = "/tmp/cert.pem"
	keyFile  = "/tmp/key.pem"
	caFile   = "/tmp/ca.pem"
)

type CertificateManager interface {
	ClientCredentials() credentials.TransportCredentials
	ServerCredentials() credentials.TransportCredentials
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

func (m *certificateManager) ClientCredentials() credentials.TransportCredentials {
	panic("implement me")
}

func (m *certificateManager) ServerCredentials() credentials.TransportCredentials {
	panic("implement me")
}

func (m *certificateManager) CertificatesUpdated(context.Context, *empty.Empty) (*empty.Empty, error) {
	logrus.Info("CertificatesUpdate request")
	if err := m.readCertificates(); err != nil {
		logrus.Error(err)
		return &empty.Empty{}, err
	}
	return &empty.Empty{}, nil
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

func (m *certificateManager) start(errorCh chan error) {
	ln, err := net.Listen("tcp", "localhost:3232")
	if err != nil {
		errorCh <- err
		return
	}
	defer ln.Close()

	s := grpc.NewServer()
	manager.RegisterManagerServer(s, m)

	if err := s.Serve(ln); err != nil {
		errorCh <- err
		return
	}
}

func NewCertificateManager() CertificateManager {
	cm := &certificateManager{}
	errorCh := make(chan error, 1)
	go cm.start(errorCh)

	return cm
}
