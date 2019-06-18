package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
	"sync"
	"time"
)

const (
	agentAddress = "/run/spire/sockets/agent.sock"
	timeout      = 5 * time.Second
)

var oidSanExtension = []int{2, 5, 29, 17}

type Manager interface {
	DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error)
	NewServer(opt ...grpc.ServerOption) (*grpc.Server, error)
	GetCertificate() *tls.Certificate
	GetCertificateBySpiffeID(spiffeID string) *x509.Certificate
	AddCertificate(certificate *x509.Certificate) error
	GetCABundle() *x509.CertPool
	GenerateJWT(spiffeID string, networkService string, obo *jwt.Token) (string, error)
	VerifyJWT(tokeString string) error
}

type CertificateObtainer interface {
	ObtainCertificates() <-chan *RetrievedCerts
	Stop()
	Error() error
}

type RetrievedCerts struct {
	keyPair  *tls.Certificate
	caBundle *x509.CertPool
}

type certificateManager struct {
	sync.RWMutex
	caBundle      *x509.CertPool
	cert          *tls.Certificate
	readyCh       chan struct{}
	crtBySpiffeID map[string]*x509.Certificate
}

func (m *certificateManager) GetCertificateBySpiffeID(spiffeID string) *x509.Certificate {
	m.RLock()
	defer m.RUnlock()
	return m.crtBySpiffeID[spiffeID]
}

func (m *certificateManager) AddCertificate(certificate *x509.Certificate) error {
	if len(certificate.DNSNames) == 0 {
		return errors.New("no extra extension with SAN")
	}

	m.Lock()
	defer m.Unlock()

	m.crtBySpiffeID[certificate.DNSNames[0]] = certificate
	logrus.Infof("certificate with spiffeID = %s added", certificate.DNSNames[0])
	return nil
}

func (m *certificateManager) DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	cred, err := m.clientCredentials()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return grpc.DialContext(ctx, target, append(opts, grpc.WithTransportCredentials(cred))...)
}

func (m *certificateManager) NewServer(opt ...grpc.ServerOption) (*grpc.Server, error) {
	cred, err := m.serverCredentials()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return grpc.NewServer(append(opt, grpc.Creds(cred), grpc.UnaryInterceptor(m.ensureValidToken))...), nil
}

func (m *certificateManager) clientCredentials() (credentials.TransportCredentials, error) {
	return credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{*m.GetCertificate()},
		RootCAs:            m.GetCABundle(),
	}), nil
}

func (m *certificateManager) serverCredentials() (credentials.TransportCredentials, error) {
	return credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{*m.GetCertificate()},
		ClientCAs:    m.GetCABundle(),
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			logrus.Info(len(rawCerts))
			c, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				logrus.Error(err)
			}
			if err := m.AddCertificate(c); err != nil {
				logrus.Error(err)
			}
			logrus.Infof("%v", c.DNSNames)
			return nil
		},
	}), nil
}

func (m *certificateManager) ensureValidToken(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	logrus.Info("ensureValidToken")
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
	}
	if err := m.VerifyJWT(md["authorization"][0]); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "token is not valid")
	}
	return handler(ctx, req)
}

func (m *certificateManager) GetCertificate() *tls.Certificate {
	logrus.Infof("Waiting for certificates...")
	<-m.readyCh
	logrus.Infof("Certificates were obtained")

	m.RLock()
	defer m.RUnlock()
	return m.cert
}

func (m *certificateManager) GetCABundle() *x509.CertPool {
	logrus.Infof("Waiting for certificates...")
	<-m.readyCh
	logrus.Infof("Certificates were obtained")

	m.RLock()
	defer m.RUnlock()
	return m.caBundle
}

func (m *certificateManager) setCertificates(c *RetrievedCerts) {
	m.Lock()
	defer m.Unlock()

	if m.cert == nil {
		close(m.readyCh)
	}
	m.cert = c.keyPair
	m.caBundle = c.caBundle
}

func (m *certificateManager) exchangeCertificates(obtainer CertificateObtainer) {
	logrus.Infof("exchangeCertificates %v", obtainer)
	certCh := obtainer.ObtainCertificates()
	logrus.Infof("ObtainCertificates() = %v", certCh)

	for {
		c, ok := <-certCh
		if ok {
			m.setCertificates(c)
			continue
		}
		if err := obtainer.Error(); err != nil {
			logrus.Errorf(err.Error())
		}
		return
	}
}

func (m *certificateManager) VerifyJWT(tokenString string) error {
	token, parts, err := new(jwt.Parser).ParseUnverified(tokenString, &nsmClaims{})
	if err != nil {
		return err
	}

	claims, ok := token.Claims.(*nsmClaims)
	if !ok {
		return errors.New("wrong claims format")
	}

	cert := m.GetCertificateBySpiffeID(claims.Subject)
	if cert == nil {
		return fmt.Errorf("no certificate proveded for %s", claims.Subject)
	}

	_, err = cert.Verify(x509.VerifyOptions{
		Roots: m.GetCABundle(),
	})

	if err != nil {
		return fmt.Errorf("certificate is signed by untrusted authority: %s", err.Error())
	}

	if err := token.Method.Verify(strings.Join(parts[0:2], "."), parts[2], cert.PublicKey); err != nil {
		return fmt.Errorf("jwt signature is not valid: %s", err.Error())
	}

	return nil
}

func (m *certificateManager) GenerateJWT(spiffeID string, networkService string, obo *jwt.Token) (string, error) {
	var oboClaim string
	if obo != nil {
		oboClaim = obo.Raw
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, &nsmClaims{
		StandardClaims: jwt.StandardClaims{
			Audience: networkService,
			Issuer:   "test",
			Subject:  spiffeID,
		},
		Obo: oboClaim,
	})

	return token.SignedString(m.GetCertificate().PrivateKey)
}

func NewManagerWithCertObtainer(obtainer CertificateObtainer) Manager {
	cm := &certificateManager{
		readyCh:       make(chan struct{}),
		crtBySpiffeID: make(map[string]*x509.Certificate),
	}
	go cm.exchangeCertificates(obtainer)
	return cm
}

func NewManager() Manager {
	obt := NewSpireCertObtainer(agentAddress, timeout)
	return NewManagerWithCertObtainer(obt)
}
