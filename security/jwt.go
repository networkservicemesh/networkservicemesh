package security

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
)

type NSMClaims struct {
	jwt.StandardClaims
	Obo  string   `json:"obo"`
	Cert []string `json:"cert"`
}

func (c *NSMClaims) getCertificate() (certs []*x509.Certificate, err error) {
	for i := 0; i < len(c.Cert); i++ {
		b, err := base64.StdEncoding.DecodeString(c.Cert[i])
		if err != nil {
			return nil, err
		}
		c, err := x509.ParseCertificate(b)
		certs = append(certs, c)
	}
	return
}

func (c *NSMClaims) verifyAndGetCertificate(caBundle *x509.CertPool) (*x509.Certificate, error) {
	crt, err := c.getCertificate()
	if err != nil {
		return nil, err
	}

	if len(crt) == 0 {
		return nil, fmt.Errorf("no certificates in chain")
	}

	if crt[0].URIs[0].String() != c.Subject {
		return nil, fmt.Errorf("spiffeID provided with JWT not equal to spiffeID from x509 TLS certificate")
	}

	interm := x509.NewCertPool()
	for i, c := range crt {
		if i == 0 {
			continue
		}
		interm.AddCert(c)
	}

	_, err = crt[0].Verify(x509.VerifyOptions{
		Roots:         caBundle,
		Intermediates: interm,
	})

	if err != nil {
		return nil, fmt.Errorf("certificate is signed by untrusted authority: %s", err.Error())
	}

	return crt[0], nil
}

func (c *NSMClaims) getObo() (*jwt.Token, []string, *NSMClaims, error) {
	if c.Obo == "" {
		return nil, nil, nil, nil
	}

	return parseJWTWithClaims(c.Obo)
}

func parseJWTWithClaims(tokenString string) (*jwt.Token, []string, *NSMClaims, error) {
	token, parts, err := new(jwt.Parser).ParseUnverified(tokenString, &NSMClaims{})
	if err != nil {
		return nil, nil, nil, err
	}

	claims, ok := token.Claims.(*NSMClaims)
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
