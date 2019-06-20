package security

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
)

type nsmClaims struct {
	jwt.StandardClaims
	Obo  string `json:"obo"`
	Cert string `json:"cert"`
}

func (c *nsmClaims) getCertificate() (*x509.Certificate, error) {
	crtBytes, err := base64.StdEncoding.DecodeString(c.Cert)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(crtBytes)
}

func (c *nsmClaims) verifyAndGetCertificate(caBundle *x509.CertPool) (*x509.Certificate, error) {
	crt, err := c.getCertificate()
	if err != nil {
		return nil, err
	}

	if crt.DNSNames[0] != c.Subject {
		return nil, fmt.Errorf("spiffeID provided with JWT not equal to spiffeID from x509 TLS certificate")
	}

	_, err = crt.Verify(x509.VerifyOptions{Roots: caBundle})
	if err != nil {
		return nil, fmt.Errorf("certificate is signed by untrusted authority: %s", err.Error())
	}

	return crt, nil
}

func (c *nsmClaims) getObo() (*jwt.Token, []string, *nsmClaims, error) {
	if c.Obo == "" {
		return nil, nil, nil, nil
	}

	return parseJWTWithClaims(c.Obo)
}

func parseJWTWithClaims(tokenString string) (*jwt.Token, []string, *nsmClaims, error) {
	token, parts, err := new(jwt.Parser).ParseUnverified(tokenString, &nsmClaims{})
	if err != nil {
		return nil, nil, nil, err
	}

	claims, ok := token.Claims.(*nsmClaims)
	if !ok {
		return nil, nil, nil, errors.New("wrong claims format")
	}

	return token, parts, claims, err
}

type NSMToken struct {
	Token string
}

func (t *NSMToken) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": t.Token,
	}, nil
}

func (t *NSMToken) RequireTransportSecurity() bool {
	return true
}
