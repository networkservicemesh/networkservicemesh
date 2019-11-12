package testsec

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/security"
)

const (
	aud            = "testaud"
	testDomain     = "test"
	spiffeIDFormat = "spiffe://%s/testID_%d"
	nameTypeURI    = 6
)

var (
	SpiffeID1 = fmt.Sprintf(spiffeIDFormat, testDomain, 1)
	SpiffeID2 = fmt.Sprintf(spiffeIDFormat, testDomain, 2)
	SpiffeID3 = fmt.Sprintf(spiffeIDFormat, testDomain, 3)
)

type testMsg struct {
	testAud string
	token   string
}

func (m *testMsg) GetSignature() string {
	return m.token
}

func testFillClaimsFunc(claims *security.ChainClaims, msg interface{}) error {
	claims.Audience = msg.(*testMsg).testAud
	return nil
}

type testSecurityProvider struct {
	ca       *x509.CertPool
	cert     *tls.Certificate
	spiffeID string
}

func newTestSecurityContext(spiffeID string) (*testSecurityProvider, error) {
	ca, err := GenerateCA()
	if err != nil {
		return nil, err
	}
	return newTestSecurityContextWithCA(spiffeID, &ca)
}

func newTestSecurityContextWithCA(spiffeID string, ca *tls.Certificate) (*testSecurityProvider, error) {
	caX509, err := x509.ParseCertificate(ca.Certificate[0])
	if err != nil {
		return nil, err
	}

	cpool := x509.NewCertPool()
	cpool.AddCert(caX509)

	cert, err := generateKeyPair(spiffeID, testDomain, ca)
	if err != nil {
		return nil, err
	}

	return &testSecurityProvider{
		ca:       cpool,
		cert:     &cert,
		spiffeID: spiffeID,
	}, nil
}

func (sc *testSecurityProvider) GetCertificate() *tls.Certificate {
	return sc.cert
}

func (sc *testSecurityProvider) GetCABundle() *x509.CertPool {
	return sc.ca
}

func (sc *testSecurityProvider) GetSpiffeID() string {
	return sc.spiffeID
}

func GenerateCA() (tls.Certificate, error) {
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

func generateKeyPair(spiffeID, domain string, caTLS *tls.Certificate) (tls.Certificate, error) {
	san, err := marshalSAN(spiffeID)
	if err != nil {
		return tls.Certificate{}, err
	}

	oidSanExtension := []int{2, 5, 29, 17}
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
			CommonName:    domain,
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

func caToBundle(caTLS *tls.Certificate) (*x509.CertPool, error) {
	caX509, err := x509.ParseCertificate(caTLS.Certificate[0])
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()
	caPool.AddCert(caX509)
	return caPool, nil
}
