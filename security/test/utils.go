package testsec

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/security"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"math/big"
	"net"
	"time"
)

var oidSanExtension = []int{2, 5, 29, 17}

const nameTypeURI = 6

type testSrv struct {
	p    security.Manager
	next string
	me   string
}

type intermediary struct {
	security.Manager
	srv        *grpc.Server
	closeFuncs []func()
}

func (s *testSrv) Request(ctx context.Context, r *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	spiffeID, ok := ctx.Value("spiffeID").(string)
	if !ok {
		return nil, errors.New("context doesn't contain spiffeID")
	}

	logrus.Infof("Receive Request, spiffeID = %s", spiffeID)
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
		client := networkservice.NewNetworkServiceClient(conn)
		if r.GetConnection().GetLabels() == nil {
			r.Connection.Labels = map[string]string{}
		}
		r.GetConnection().GetLabels()[s.me] = spiffeID

		response, err := client.Request(ctx, r)
		if err != nil {
			return nil, err
		}

		err = s.p.SignResponse(response, response.GetResponseJWT())
		if err != nil {
			return nil, err
		}

		return response, nil
	}

	if r.GetConnection() == nil {
		r.Connection = &connection.Connection{}
	}

	if r.GetConnection().GetLabels() == nil {
		r.Connection.Labels = map[string]string{}
	}
	r.GetConnection().GetLabels()[s.me] = spiffeID
	response := &connection.Connection{
		Id:     "testId",
		Labels: r.GetConnection().GetLabels(),
	}
	s.p.SignResponse(response, "")
	return response, nil
}

func emptyRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Labels: map[string]string{},
		},
	}
}

func (s *testSrv) Close(context.Context, *connection.Connection) (*empty.Empty, error) {
	panic("implement me")
}

func createSimpleIntermediary(spiffeID string, port int, ca *tls.Certificate, next, name string) (*intermediary, error) {
	obt, err := newSimpleCertObtainerWithCA(spiffeID, ca)
	if err != nil {
		return nil, err
	}

	return createIntermediaryWithObt(obt, port, next, name)
}

func createExchangeIntermediary(spiffeID string, port int, ca *tls.Certificate, next, name string) (*intermediary, error) {
	obt, err := newExchangeCertObtainerWithCA(spiffeID, ca, 3*time.Second)
	if err != nil {
		return nil, err
	}

	return createIntermediaryWithObt(obt, port, next, name)
}

func createIntermediaryWithObt(obt security.CertificateObtainer, port int, next, name string) (*intermediary, error) {
	mgr := security.NewManagerWithCertObtainer(obt)
	srv := mgr.NewServer()

	networkservice.RegisterNetworkServiceServer(srv, &testSrv{
		p:    mgr,
		next: next,
		me:   name,
	})

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	go srv.Serve(ln)

	return &intermediary{
		Manager: mgr,
		srv:     srv,
		closeFuncs: []func(){
			func() { _ = ln.Close() },
		},
	}, nil
}

func (p *intermediary) NewClient(target string) (networkservice.NetworkServiceClient, error) {
	conn, err := p.DialContext(context.Background(), target)
	if err != nil {
		return nil, err
	}
	p.closeFuncs = append(p.closeFuncs, func() { _ = conn.Close() })
	return networkservice.NewNetworkServiceClient(conn), nil
}

func (p *intermediary) Close() {
	for _, f := range p.closeFuncs {
		f()
	}
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
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		BasicConstraintsValid: true,
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	pub := &priv.PublicKey

	certBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, pub, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
	return tls.X509KeyPair(certPem, keyPem)
}

func marshalSAN(spiffeID string) ([]byte, error) {
	return asn1.Marshal([]asn1.RawValue{{Tag: nameTypeURI, Class: 2, Bytes: []byte(spiffeID)}})
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
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		KeyUsage:           x509.KeyUsageDigitalSignature,
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	pub := &priv.PublicKey

	ca, err := x509.ParseCertificate(caTLS.Certificate[0])
	if err != nil {
		return tls.Certificate{}, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, pub, caTLS.PrivateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
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

	return security.NewManagerWithCertObtainer(obt), nil
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
