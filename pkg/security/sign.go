package security

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"time"
)

type Signed interface {
	GetSignature() string
}

type ClaimsSetter func(claims *ChainClaims, msg interface{}) error

type SignOption interface {
	apply(*ChainClaims)
}

type funcSignOption struct {
	f func(*ChainClaims)
}

func (fso *funcSignOption) apply(cc *ChainClaims) {
	fso.f(cc)
}

func newFuncSignOption(f func(*ChainClaims)) SignOption {
	return &funcSignOption{
		f: f,
	}
}

func WithObo(obo string) SignOption {
	return newFuncSignOption(func(claims *ChainClaims) {
		claims.Obo = obo
	})
}

func WithLifetime(t time.Duration) SignOption {
	return newFuncSignOption(func(claims *ChainClaims) {
		claims.ExpiresAt = time.Now().Add(t).Unix()
	})
}

func GenerateSignature(msg interface{}, claimsSetter ClaimsSetter, p Provider, opts ...SignOption) (string, error) {
	claims := &ChainClaims{}
	claimsSetter(claims, msg)

	for _, o := range opts {
		o.apply(claims)
	}

	if claims.Obo != "" {
		token, parts, oboClaims, err := ParseJWTWithClaims(claims.Obo)
		if err != nil {
			return "", err
		}

		if err := verifyJWTChain(token, parts, oboClaims, p.GetCABundle()); err != nil {
			return "", fmt.Errorf("obo token validation error: %s", err.Error())
		}

		if oboClaims.Subject == p.GetSpiffeID() {
			return claims.Obo, nil
		}
	}

	var certs []string
	for i := 0; i < len(p.GetCertificate().Certificate); i++ {
		certs = append(certs, base64.StdEncoding.EncodeToString(p.GetCertificate().Certificate[i]))
	}

	claims.Subject = p.GetSpiffeID()
	claims.Cert = certs

	return jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(p.GetCertificate().PrivateKey)
}

func VerifySignature(signature string, ca *x509.CertPool, spiffeID string) error {
	token, parts, claims, err := ParseJWTWithClaims(signature)
	if err != nil {
		return err
	}

	if claims.Subject != spiffeID {
		return fmt.Errorf("wrong spiffeID")
	}

	if claims.ExpiresAt != 0 {
		if time.Now().After(time.Unix(claims.ExpiresAt, 0)) {
			return fmt.Errorf("token expired")
		}
	}

	return verifyJWTChain(token, parts, claims, ca)
}
