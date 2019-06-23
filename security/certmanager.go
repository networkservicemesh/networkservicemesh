package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"reflect"
	"strings"
	"sync"
	"time"
)

const (
	agentAddress = "/run/spire/sockets/agent.sock"
	timeout      = 5 * time.Second
)

type Manager interface {
	DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error)
	NewServer(opt ...grpc.ServerOption) (*grpc.Server, error)
	GetCertificate() *tls.Certificate
	GetCABundle() *x509.CertPool
	GenerateJWT(networkService string, obo string) (string, error)
	VerifyJWT(spiffeID, tokeString string) error
}

type CertificateObtainer interface {
	ObtainCertificates() <-chan *RetrievedCerts
	Stop()
	Error() error
}

type RetrievedCerts struct {
	TLSCert  *tls.Certificate
	CABundle *x509.CertPool
}

type certificateManager struct {
	sync.RWMutex
	caBundle      *x509.CertPool
	cert          *tls.Certificate
	readyCh       chan struct{}
	crtBySpiffeID map[string]*x509.Certificate
	audByReqFunc  func(req interface{}) (string, error)
}

func (m *certificateManager) DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	cred, err := m.clientCredentials()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return grpc.DialContext(ctx, target, append(opts, grpc.WithTransportCredentials(cred), grpc.WithUnaryInterceptor(m.clientInterceptor))...)
}

func (m *certificateManager) NewServer(opt ...grpc.ServerOption) (*grpc.Server, error) {
	cred, spiffeIDFunc, err := m.serverCredentials()
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	validFunc := m.makeJwtTokenValidator(spiffeIDFunc)
	return grpc.NewServer(append(opt, grpc.Creds(cred), grpc.UnaryInterceptor(validFunc))...), nil
}

func (m *certificateManager) clientCredentials() (credentials.TransportCredentials, error) {
	return credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{*m.GetCertificate()},
		RootCAs:            m.GetCABundle(),
	}), nil
}

func (m *certificateManager) serverCredentials() (credentials.TransportCredentials, func() string, error) {
	spiffeIDCh := make(chan string, 1)

	return credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{*m.GetCertificate()},
		ClientCAs:    m.GetCABundle(),
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			c, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				logrus.Error(err)
			}
			spiffeIDCh <- c.URIs[0].String()
			logrus.Infof("%v", c.URIs)
			return nil
		},
	}), spiffeID(spiffeIDCh), nil
}

func spiffeID(spiffeIDCh <-chan string) func() string {
	var spiffeIDCache string
	return func() string {
		if spiffeIDCache != "" {
			return spiffeIDCache
		}
		spiffeIDCache = <-spiffeIDCh
		return spiffeIDCache
	}
}

func (m *certificateManager) clientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	logrus.Infof("ClientInterceptor start working...")
	aud, err := m.audByReqFunc(req)
	if err != nil {
		logrus.Error(err)
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	var obo string
	if ctx.Value("obo") != nil {
		obo = ctx.Value("obo").(string)
	}

	jwt, err := m.GenerateJWT(aud, obo)
	if err != nil {
		logrus.Error(err)
		return err
	}

	return invoker(ctx, method, req, reply, cc, append(opts, grpc.PerRPCCredentials(&NSMToken{Token: jwt}))...)
}

func (m *certificateManager) makeJwtTokenValidator(spiffeIDFunc func() string) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}

		if len(md["authorization"]) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "no token provided")
		}

		jwt := md["authorization"][0]
		if err := m.VerifyJWT(spiffeIDFunc(), jwt); err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "token is not valid")
		}

		_, _, claims, _ := parseJWTWithClaims(jwt)

		newCtx := context.WithValue(ctx, "spiffeID", spiffeIDFunc())
		newCtx = context.WithValue(newCtx, "obo", jwt)
		newCtx = context.WithValue(newCtx, "aud", claims.Audience)
		return handler(newCtx, req)
	}
}

func (m *certificateManager) GetCertificate() *tls.Certificate {
	<-m.readyCh

	m.RLock()
	defer m.RUnlock()
	return m.cert
}

func (m *certificateManager) GetCABundle() *x509.CertPool {
	<-m.readyCh

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
	m.cert = c.TLSCert
	m.caBundle = c.CABundle
}

func (m *certificateManager) exchangeCertificates(obtainer CertificateObtainer) {
	certCh := obtainer.ObtainCertificates()

	for {
		c, ok := <-certCh
		if ok {
			logrus.Info("New x509 certificate obtained")
			m.setCertificates(c)
			continue
		}
		if err := obtainer.Error(); err != nil {
			logrus.Errorf(err.Error())
		}
		return
	}
}

func (m *certificateManager) VerifyJWT(transportSpiffeID, tokenString string) error {
	token, parts, claims, err := parseJWTWithClaims(tokenString)
	if err != nil {
		return err
	}

	if claims.Subject != transportSpiffeID {
		return fmt.Errorf("wrong spiffeID")
	}

	return m.verifyJWTChain(token, parts, claims)
}

func (m *certificateManager) verifyJWTChain(token *jwt.Token, parts []string, claims *nsmClaims) error {
	currentToken, currentParts, currentClaims := token, parts, claims

	for currentToken != nil {
		err := m.verifyJwt(currentToken, currentParts, currentClaims)
		if err != nil {
			return err
		}

		currentToken, currentParts, currentClaims, err = currentClaims.getObo()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *certificateManager) verifyJwt(token *jwt.Token, parts []string, claims *nsmClaims) error {
	logrus.Infof("Validating JWT: %s", claims.Subject)
	crt, err := claims.verifyAndGetCertificate(m.GetCABundle())
	if err != nil {
		return err
	}

	if err := token.Method.Verify(strings.Join(parts[0:2], "."), parts[2], crt.PublicKey); err != nil {
		return fmt.Errorf("jwt signature is not valid: %s", err.Error())
	}

	return nil
}

func (m *certificateManager) GenerateJWT(networkService string, obo string) (string, error) {
	crtBytes := m.GetCertificate().Certificate[0]
	x509crt, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		return "", err
	}

	if obo != "" {
		token, parts, claims, err := parseJWTWithClaims(obo)
		if err != nil {
			return "", err
		}

		if err := m.verifyJwt(token, parts, claims); err != nil {
			return "", fmt.Errorf("obo token validation error: %s", err.Error())
		}

		if claims.Subject == x509crt.URIs[0].String() {
			return obo, nil
		}
	}

	certStr := base64.StdEncoding.EncodeToString(crtBytes)

	token := jwt.NewWithClaims(jwt.SigningMethodES256, &nsmClaims{
		StandardClaims: jwt.StandardClaims{
			Audience: networkService,
			Issuer:   "test",
			Subject:  x509crt.URIs[0].String(),
		},
		Obo:  obo,
		Cert: certStr,
	})
	logrus.Infof("m.GetCertificate().PrivateKey.Type = %v", reflect.TypeOf(m.GetCertificate().PrivateKey))
	return token.SignedString(m.GetCertificate().PrivateKey)
}

func NewManagerWithCertObtainer(obtainer CertificateObtainer, audByReqFunc func(req interface{}) (string, error)) Manager {
	cm := &certificateManager{
		readyCh:       make(chan struct{}),
		crtBySpiffeID: make(map[string]*x509.Certificate),
		audByReqFunc:  audByReqFunc,
	}
	go cm.exchangeCertificates(obtainer)
	return cm
}

func networserviceAud(req interface{}) (string, error) {
	if r, ok := req.(*networkservice.NetworkServiceRequest); ok {
		return r.GetConnection().GetNetworkService(), nil
	}
	if r, ok := req.(*registry.NSERegistration); ok {
		return r.GetNetworkService().GetName(), nil
	}

	return "", errors.New("unable to get networkservice from request")
}

func NewManager() Manager {
	obt := NewSpireCertObtainer(agentAddress, timeout)
	return NewManagerWithCertObtainer(obt, networserviceAud)
}
