package security

import (
	"context"
	"crypto/x509"
	"encoding/base64"

	"github.com/pkg/errors"

	"github.com/dgrijalva/jwt-go"
)

type contextKey int

const (
	NSMClaimsContextKey contextKey = iota
)

// NSMToken is implementation of PerRPCCredentials for NSM
type NSMToken struct {
	Token string
}

// GetRequestMetadata implements methods from PerRPCCredentials
func (t *NSMToken) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": t.Token,
	}, nil
}

// RequireTransportSecurity implements methods from PerRPCCredentials
func (t *NSMToken) RequireTransportSecurity() bool {
	return true
}

// ChainClaims jwt claims for NSM token
type ChainClaims struct {
	jwt.StandardClaims
	Obo  string   `json:"obo"`
	Cert []string `json:"cert"`
}

func (c *ChainClaims) getCertificate() (certs []*x509.Certificate, err error) {
	for i := 0; i < len(c.Cert); i++ {
		b, err := base64.StdEncoding.DecodeString(c.Cert[i])
		if err != nil {
			return nil, err
		}
		c, err := x509.ParseCertificate(b)
		if err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return
}

func (c *ChainClaims) verifyAndGetCertificate(caBundle *x509.CertPool) (*x509.Certificate, error) {
	crt, err := c.getCertificate()
	if err != nil {
		return nil, err
	}

	if len(crt) == 0 {
		return nil, errors.New("no certificates in chain")
	}

	if crt[0].URIs[0].String() != c.Subject {
		return nil, errors.New("spiffeID provided with JWT not equal to spiffeID from x509 TLS certificate")
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
		return nil, errors.Wrap(err, "certificate is signed by untrusted authority: %s")
	}

	return crt[0], nil
}

func (c *ChainClaims) parseObo() (*jwt.Token, []string, *ChainClaims, error) {
	if c.Obo == "" {
		return nil, nil, nil, nil
	}

	return ParseJWTWithClaims(c.Obo)
}
