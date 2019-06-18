package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"math/big"
	"net"
	"testing"
	"time"
)

const (
	testDomain   = "test.com"
	testSpiffeID = "spiffe://test.com/test"
)

type testSimpleCertificateObtainer struct {
	cert *RetrievedCerts
}

func newTestCertificateObtainer() (CertificateObtainer, error) {
	ca, err := generateCA()
	if err != nil {
		return nil, err
	}

	return newTestCertificateObtainerWithCA(ca)
}

func newTestCertificateObtainerWithCA(caTLS tls.Certificate) (CertificateObtainer, error) {
	caX509, err := x509.ParseCertificate(caTLS.Certificate[0])
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caX509)

	crt, err := generateKeyPair(testSpiffeID, &caTLS)
	if err != nil {
		return nil, err
	}

	return &testSimpleCertificateObtainer{
		cert: &RetrievedCerts{
			caBundle: caPool,
			keyPair:  &crt,
		},
	}, nil
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

func (t *testSimpleCertificateObtainer) ObtainCertificates() <-chan *RetrievedCerts {
	certCh := make(chan *RetrievedCerts, 1)
	certCh <- t.cert
	close(certCh)
	return certCh
}

func (*testSimpleCertificateObtainer) Stop() {
}

func (*testSimpleCertificateObtainer) Error() error {
	return nil
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

func TestSimpleCertCreation(t *testing.T) {
	RegisterTestingT(t)

	ca, err := generateCA()
	Expect(err).To(BeNil())

	caX509, err := x509.ParseCertificate(ca.Certificate[0])
	Expect(err).To(BeNil())

	roots := x509.NewCertPool()
	roots.AddCert(caX509)

	crt, err := generateKeyPair(testSpiffeID, &ca)
	Expect(err).To(BeNil())

	err = verify(&crt, roots)
	Expect(err).To(BeNil())
}

func TestSimpleCertObtainer(t *testing.T) {
	RegisterTestingT(t)

	obt, err := newTestCertificateObtainer()
	Expect(err).To(BeNil())

	mgr := NewManagerWithCertObtainer(obt)

	crt := mgr.GetCertificate()
	ca := mgr.GetCABundle()
	verify(crt, ca)
}

type helloSrv struct{}

func (hs *helloSrv) SayHello(ctx context.Context, r *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: r.GetName()}, nil
}

func TestClientServer(t *testing.T) {
	RegisterTestingT(t)

	obt, err := newTestCertificateObtainer()
	Expect(err).To(BeNil())

	mgr := NewManagerWithCertObtainer(obt)

	srv, err := mgr.NewServer()
	Expect(err).To(BeNil())

	helloworld.RegisterGreeterServer(srv, &helloSrv{})

	ln, err := net.Listen("tcp", ":3434")
	Expect(err).To(BeNil())
	defer ln.Close()
	go srv.Serve(ln)

	secureConn, err := mgr.DialContext(context.Background(), ":3434")
	Expect(err).To(BeNil())
	defer secureConn.Close()

	gc := helloworld.NewGreeterClient(secureConn)
	response, err := gc.SayHello(context.Background(), &helloworld.HelloRequest{Name: "testName"})
	Expect(err).To(BeNil())
	Expect(response.Message).To(Equal("testName"))

	_, err = grpc.DialContext(context.Background(), ":3434")
	Expect(err).ToNot(BeNil())
}
