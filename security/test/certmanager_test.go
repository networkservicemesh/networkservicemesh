package test

import (
	"crypto/x509"
	"github.com/dgrijalva/jwt-go"
	"github.com/networkservicemesh/networkservicemesh/security"
	. "github.com/onsi/gomega"
	"strings"
	"testing"
)

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

	obt, err := newSimpleCertObtainer(testSpiffeID)
	Expect(err).To(BeNil())

	mgr := security.NewManagerWithCertObtainer(obt)

	crt := mgr.GetCertificate()
	ca := mgr.GetCABundle()
	verify(crt, ca)
}

func TestTamperToken(t *testing.T) {
	RegisterTestingT(t)

	obt, err := newSimpleCertObtainer(testSpiffeID)
	Expect(err).To(BeNil())

	mgr := security.NewManagerWithCertObtainer(obt)
	token, err := mgr.GenerateJWT("123", "")
	Expect(err).To(BeNil())

	ptoken, parts, err := new(jwt.Parser).ParseUnverified(token, &security.NSMClaims{})
	ptoken.Claims.(*security.NSMClaims).Audience = "hacked"

	ss, err := ptoken.SigningString()
	Expect(err).To(BeNil())

	hack := strings.Join([]string{ss, parts[2]}, ".")

	err = mgr.VerifyJWT(testSpiffeID, hack)
	Expect(err).ToNot(BeNil())
}
