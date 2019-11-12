package security

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"gopkg.in/square/go-jose.v2"
	"strings"
)

type Signature struct {
	Token  *jwt.Token
	Parts  []string
	Claims *ChainClaims
	JWKS   *jose.JSONWebKeySet
}

func (s *Signature) GetSpiffeID() string {
	if s.Claims == nil {
		return ""
	}
	return s.Claims.Subject
}

func (s *Signature) ToString() (string, error) {
	if s.Token == nil {
		return "", errors.New("Token is empty")
	}

	if s.JWKS == nil {
		return "", errors.New("JWKS is empty")
	}

	return SignatureString(s.Token.Raw, s.JWKS)
}

func SignatureString(jwt string, jwks *jose.JSONWebKeySet) (string, error) {
	b, err := json.Marshal(jwks)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", jwt, base64.StdEncoding.EncodeToString(b)), nil
}

func (s *Signature) Parse(signature string) error {
	strs := strings.Split(signature, ":")
	if len(strs) != 2 {
		return errors.New("token with JWKS in bad format")
	}

	b, err := base64.StdEncoding.DecodeString(strs[1])
	if err != nil {
		return err
	}

	jwks := &jose.JSONWebKeySet{}
	if err := json.Unmarshal(b, jwks); err != nil {
		return err
	}
	s.JWKS = jwks

	s.Token, s.Parts, s.Claims, err = ParseJWTWithClaims(strs[0])
	return err
}
