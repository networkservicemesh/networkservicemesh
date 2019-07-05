package security

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/networkservice"
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

type Manager interface {
	DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error)
	NewServer(opts ...grpc.ServerOption) *grpc.Server
	GetCertificate() *tls.Certificate
	GetCABundle() *x509.CertPool
	GenerateJWT(networkService string, obo string) (string, error)
	VerifyJWT(spiffeID, tokeString string) error
	SignResponse(resp interface{}, obo string) error
	GetSpiffeID() string
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
}

var once sync.Once
var manager Manager

func GetSecurityManager() Manager {
	logrus.Info("Getting SecurityManager...")
	once.Do(func() {
		logrus.Info("Creating new SecurityManager...")
		manager = NewManager()
	})
	return manager
}

func InitSecurityManagerWithExisting(mgr Manager) {
	logrus.Info("Initializing Security Manager with existing one...")
	once.Do(func() {
		manager = mgr
	})
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

func (m *certificateManager) DialContext(ctx context.Context, target string, opts ...grpc.DialOption) (conn *grpc.ClientConn, err error) {
	cred, spiffeIDFunc := m.clientCredentials()

	opts = append(opts,
		grpc.WithTransportCredentials(cred),
		grpc.WithUnaryInterceptor(m.createClientInterceptor(spiffeIDFunc)))

	return grpc.DialContext(ctx, target, opts...)
}

func (m *certificateManager) NewServer(opts ...grpc.ServerOption) *grpc.Server {
	cred, spiffeIDFunc := m.serverCredentials()
	opts = append(opts,
		grpc.Creds(cred),
		grpc.UnaryInterceptor(m.createServerInterceptor(spiffeIDFunc)))

	return grpc.NewServer(opts...)
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

func (m *certificateManager) GetSpiffeID() string {
	crtBytes := m.GetCertificate().Certificate[0]
	x509crt, err := x509.ParseCertificate(crtBytes)
	if err != nil {
		logrus.Error(err)
		return ""
	}
	return x509crt.URIs[0].String()
}

func (m *certificateManager) GenerateJWT(networkService string, obo string) (string, error) {
	spiffeID := m.GetSpiffeID()

	if obo != "" {
		token, parts, claims, err := parseJWTWithClaims(obo)
		if err != nil {
			return "", err
		}

		if err := m.verifyJwt(token, parts, claims); err != nil {
			return "", fmt.Errorf("obo token validation error: %s", err.Error())
		}

		if claims.Subject == spiffeID {
			return obo, nil
		}
	}

	var certs []string
	for i := 0; i < len(m.GetCertificate().Certificate); i++ {
		certs = append(certs, base64.StdEncoding.EncodeToString(m.GetCertificate().Certificate[i]))
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, &NSMClaims{
		StandardClaims: jwt.StandardClaims{
			Audience:  networkService,
			Issuer:    "test",
			Subject:   spiffeID,
			ExpiresAt: time.Now().Add(2 * time.Second).Unix(),
		},
		Obo:  obo,
		Cert: certs,
	})

	return token.SignedString(m.GetCertificate().PrivateKey)
}

func (m *certificateManager) VerifyJWT(transportSpiffeID, tokenString string) error {
	token, parts, claims, err := parseJWTWithClaims(tokenString)
	if err != nil {
		return err
	}

	if claims.Subject != transportSpiffeID {
		return fmt.Errorf("wrong spiffeID")
	}

	if claims.ExpiresAt != 0 {
		if time.Now().After(time.Unix(claims.ExpiresAt, 0)) {
			return fmt.Errorf("token expired")
		}
	}

	return m.verifyJWTChain(token, parts, claims)
}

func (m *certificateManager) SignResponse(resp interface{}, obo string) error {
	conn, ok := resp.(connection.Connection)
	if !ok {
		return errors.New("unable to sign response: unsupported type")
	}

	responseJWT, err := m.GenerateJWT(conn.GetNetworkService(), obo)
	if err != nil {
		return err
	}

	conn.SetResponseJWT(responseJWT)
	return nil
}

func cachingReadFirst(ch chan string) func() string {
	var cache string
	cached := false
	return func() string {
		if cached {
			return cache
		}
		cache = <-ch
		cached = true
		return cache
	}
}

func parseSpiffeID(rawCert []byte) (string, error) {
	crt, err := x509.ParseCertificate(rawCert)
	if err != nil {
		return "", err
	}
	return crt.URIs[0].String(), nil
}

func (m *certificateManager) clientCredentials() (credentials.TransportCredentials, func() string) {
	spiffeIDCh := make(chan string, 1)

	return credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{*m.GetCertificate()},
		RootCAs:            m.GetCABundle(),
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("no certificates from peer")
			}
			spiffeID, err := parseSpiffeID(rawCerts[0])
			if err != nil {
				return err
			}
			spiffeIDCh <- spiffeID
			logrus.Infof("%v", spiffeID)
			return nil
		},
	}), cachingReadFirst(spiffeIDCh)
}

func (m *certificateManager) serverCredentials() (credentials.TransportCredentials, func() string) {
	spiffeIDCh := make(chan string, 1)

	return credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{*m.GetCertificate()},
		ClientCAs:    m.GetCABundle(),
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("no certificates from peer")
			}
			spiffeID, err := parseSpiffeID(rawCerts[0])
			if err != nil {
				return err
			}
			spiffeIDCh <- spiffeID
			logrus.Infof("%v", spiffeID)
			return nil
		},
	}), cachingReadFirst(spiffeIDCh)
}

func (m *certificateManager) createClientInterceptor(spiffeIDFunc func() string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		request, ok := req.(networkservice.Request)
		if !ok {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		logrus.Infof("ClientInterceptor start working...")

		var obo string
		if ctx.Value("obo") != nil {
			obo = ctx.Value("obo").(string)
		}

		token, err := m.GenerateJWT(request.GetRequestConnection().GetNetworkService(), obo)
		if err != nil {
			logrus.Error(err)
			return err
		}

		if err := invoker(ctx, method, req, reply, cc, append(opts, grpc.PerRPCCredentials(&NSMToken{Token: token}))...); err != nil {
			return err
		}

		conn, ok := reply.(connection.Connection)
		if !ok {
			return errors.New("can't verify response: wrong type")
		}

		if err := m.VerifyJWT(spiffeIDFunc(), conn.GetResponseJWT()); err != nil {
			return status.Errorf(codes.Unauthenticated, "response jwt is not valid")
		}

		return nil
	}
}

func (m *certificateManager) createServerInterceptor(spiffeIDFunc func() string) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		_, ok := req.(networkservice.Request)
		if !ok {
			return handler(ctx, req)
		}

		logrus.Infof("ServerInterceptor start working...")
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}

		if len(md["authorization"]) == 0 {
			return nil, status.Errorf(codes.Unauthenticated, "no token provided")
		}

		jwt := md["authorization"][0]
		if err := m.VerifyJWT(spiffeIDFunc(), jwt); err != nil {
			return nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("token is not valid: %v", err))
		}

		_, _, claims, _ := parseJWTWithClaims(jwt)

		newCtx := context.WithValue(ctx, "spiffeID", spiffeIDFunc())
		newCtx = context.WithValue(newCtx, "obo", jwt)
		newCtx = context.WithValue(newCtx, "aud", claims.Audience)

		return handler(newCtx, req)
	}
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

func (m *certificateManager) verifyJWTChain(token *jwt.Token, parts []string, claims *NSMClaims) error {
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

func (m *certificateManager) verifyJwt(token *jwt.Token, parts []string, claims *NSMClaims) error {
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
