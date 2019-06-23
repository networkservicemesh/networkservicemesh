package test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/security"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"math/big"
	"net"
	"time"
)

var oidSanExtension = []int{2, 5, 29, 17}

type helloSrv struct {
	p    *party
	next string
	me   string
}

type party struct {
	security.Manager
	srv        *grpc.Server
	client     helloworld.GreeterClient
	closeFuncs []func()
}

func helloworldAud(req interface{}) (string, error) {
	r, ok := req.(*helloworld.HelloRequest)
	if !ok {
		return "", errors.New("request does not have type helloworld.HelloRequest")
	}
	return r.GetName(), nil
}

func createGreeterParty(spiffeID string, port int, ca *tls.Certificate, next, name string) (*party, error) {
	rv, err := createParty(spiffeID, port, ca)
	if err != nil {
		return nil, err
	}

	helloworld.RegisterGreeterServer(rv.srv, &helloSrv{
		p:    rv,
		next: next,
		me:   name,
	})

	return rv, nil
}

func createParty(spiffeID string, port int, ca *tls.Certificate) (*party, error) {
	rv := &party{}

	obt, err := newSimpleCertObtainerWithCA(spiffeID, ca)
	if err != nil {
		return nil, err
	}

	rv.Manager = security.NewManagerWithCertObtainer(obt, helloworldAud)
	rv.srv, err = rv.NewServer()
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	go rv.srv.Serve(ln)

	rv.closeFuncs = []func(){
		func() { _ = ln.Close() },
	}
	return rv, nil
}

func (p *party) SayHello(msg, target, obo string) (*helloworld.HelloReply, error) {
	if p.client == nil {
		conn, err := p.DialContext(context.Background(), target)
		if err != nil {
			return nil, err
		}
		p.closeFuncs = append(p.closeFuncs, func() { _ = conn.Close() })
		p.client = helloworld.NewGreeterClient(conn)
	}

	request := &helloworld.HelloRequest{Name: msg}
	return p.client.SayHello(context.Background(), request)
}

func (p *party) Close() {
	for _, f := range p.closeFuncs {
		f()
	}
}

func (s *helloSrv) SayHello(ctx context.Context, r *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	logrus.Infof("Receive SayHello, spiffeID = %s", ctx.Value("spiffeID"))
	logrus.Infof("obo = %s", ctx.Value("obo"))
	logrus.Infof("aud = %s", ctx.Value("aud"))
	logrus.Infof("next = %s", s.next)
	logrus.Infof("me = %s", s.me)

	if s.next != "" {
		conn, err := s.p.DialContext(context.Background(), s.next)
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		client := helloworld.NewGreeterClient(conn)
		request := &helloworld.HelloRequest{Name: r.Name + s.me}

		return client.SayHello(ctx, request)
	}

	return &helloworld.HelloReply{Message: r.GetName() + s.me}, nil
}

type testConnectionMonitor struct {
}

func (cm *testConnectionMonitor) MonitorConnections(empty *empty.Empty, stream connection.MonitorConnection_MonitorConnectionsServer) error {
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("%d", i)
		stream.Send(&connection.ConnectionEvent{
			Type: connection.ConnectionEventType_UPDATE,
			Connections: map[string]*connection.Connection{
				id: {
					Id:             id,
					NetworkService: "ns",
				},
			},
		})
	}

	return nil
}

func generateCA() (tls.Certificate, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1653),
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	pub := &priv.PublicKey

	certBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, pub, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return tls.X509KeyPair(certPem, keyPem)
}

func marshalSAN(spiffeID string) ([]byte, error) {
	return asn1.Marshal([]asn1.RawValue{{Tag: 2, Class: 2, Bytes: []byte(spiffeID)}})
}

func generateKeyPair(spiffeID string, caTLS *tls.Certificate) (tls.Certificate, error) {
	san, err := marshalSAN(spiffeID)
	if err != nil {
		return tls.Certificate{}, err
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
			CommonName:    testDomain,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		ExtraExtensions: []pkix.Extension{
			{
				Id:    oidSanExtension,
				Value: san,
			},
		},
		KeyUsage: x509.KeyUsageDigitalSignature,
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}
	pub := &priv.PublicKey

	ca, err := x509.ParseCertificate(caTLS.Certificate[0])
	if err != nil {
		return tls.Certificate{}, err
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, pub, caTLS.PrivateKey)

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	return tls.X509KeyPair(certPem, keyPem)
}

func verify(crt *tls.Certificate, caBundle *x509.CertPool) error {
	crtX509, err := x509.ParseCertificate(crt.Certificate[0])
	if err != nil {
		return err
	}
	_, err = crtX509.Verify(x509.VerifyOptions{
		Roots: caBundle,
	})
	return err
}

func createPartyWithCA(spiffeID string, caTLS tls.Certificate) (security.Manager, error) {
	obt, err := newSimpleCertObtainerWithCA(spiffeID, &caTLS)
	if err != nil {
		return nil, err
	}

	return security.NewManagerWithCertObtainer(obt, helloworldAud), nil
}

func createParties() (security.Manager, security.Manager, error) {
	ca, err := generateCA()
	if err != nil {
		return nil, nil, err
	}

	p1, err := createPartyWithCA(testSpiffeID, ca)
	if err != nil {
		return nil, nil, err
	}

	p2, err := createPartyWithCA(testSpiffeID, ca)
	if err != nil {
		return nil, nil, err
	}

	return p1, p2, nil
}

func caToBundle(caTLS *tls.Certificate) (*x509.CertPool, error) {
	caX509, err := x509.ParseCertificate(caTLS.Certificate[0])
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caX509)
	return caPool, nil
}
