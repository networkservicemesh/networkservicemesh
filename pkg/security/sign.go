// Copyright (c) 2019 Cisco Systems, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package security

import (
	"crypto/x509"
	"strings"
	"time"

	"gopkg.in/square/go-jose.v2"

	"github.com/pkg/errors"

	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

type Signed interface {
	GetSignature() string
	SetSignature(sign string)
}

type ClaimsSetter func(claims *ChainClaims, msg interface{}) error

type SignOption interface {
	apply(*signConfig)
}

type signConfig struct {
	obo       *Signature
	expiresAt time.Duration
}

type funcSignOption struct {
	f func(config *signConfig)
}

func (fso *funcSignOption) apply(cfg *signConfig) {
	fso.f(cfg)
}

func newFuncSignOption(f func(*signConfig)) SignOption {
	return &funcSignOption{
		f: f,
	}
}

func WithObo(obo *Signature) SignOption {
	return newFuncSignOption(func(cfg *signConfig) {
		cfg.obo = obo
	})
}

func WithLifetime(t time.Duration) SignOption {
	return newFuncSignOption(func(cfg *signConfig) {
		cfg.expiresAt = t
	})
}

func GenerateSignature(msg interface{}, claimsSetter ClaimsSetter, p Provider, opts ...SignOption) (string, error) {
	cfg := &signConfig{}
	for _, o := range opts {
		o.apply(cfg)
	}

	claims := &ChainClaims{}
	if err := claimsSetter(claims, msg); err != nil {
		return "", err
	}

	if cfg.obo != nil && cfg.obo.GetSpiffeID() == p.GetSpiffeID() {
		logrus.Info("GeneratingSignature: claims.Obo.Subject equals current SpiffeID")
		return cfg.obo.ToString()
	}

	if cfg.obo != nil && cfg.obo.Token != nil {
		logrus.Info("GeneratingSignature: claims.Obo is not empty")
		claims.Obo = cfg.obo.Token.Raw
	}

	if cfg.expiresAt != 0 {
		claims.ExpiresAt = time.Now().Add(cfg.expiresAt).Unix()
	}

	var xcerts []*x509.Certificate

	for _, c := range p.GetCertificate().Certificate {
		crt, err := x509.ParseCertificate(c)
		if err != nil {
			return "", err
		}
		xcerts = append(xcerts, crt)
	}

	if len(xcerts) == 0 {
		return "", errors.New("certificate list is empty")
	}

	jwks := &jose.JSONWebKeySet{}
	if cfg.obo != nil && cfg.obo.JWKS != nil {
		jwks.Keys = cfg.obo.JWKS.Keys
	}

	jwks.Keys = append(jwks.Keys, jose.JSONWebKey{
		KeyID:        p.GetSpiffeID(),
		Key:          xcerts[0].PublicKey,
		Certificates: xcerts,
	})

	claims.Subject = p.GetSpiffeID()

	token, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(p.GetCertificate().PrivateKey)
	if err != nil {
		return "", err
	}

	return SignatureString(token, jwks)
}

func VerifySignature(signature string, ca *x509.CertPool, spiffeID string) error {
	s := &Signature{}
	err := s.Parse(signature)
	if err != nil {
		return err
	}

	return verifySignatureParsed(s, ca, spiffeID)
}

func verifySignatureParsed(s *Signature, ca *x509.CertPool, spiffeID string) error {
	if s.Claims.Subject != spiffeID {
		return errors.New("wrong spiffeID")
	}

	if s.Claims.ExpiresAt != 0 {
		if time.Now().After(time.Unix(s.Claims.ExpiresAt, 0)) {
			return errors.New("token expired")
		}
	}

	return verifyChainJWT(s, ca)
}

func verifyChainJWT(s *Signature, ca *x509.CertPool) error {
	current := s

	for current != nil {
		err := verifySingleJWT(current, ca)
		if err != nil {
			return err
		}

		if current.Claims != nil && current.Claims.Obo == "" {
			return nil
		}

		token, parts, claims, err := ParseJWTWithClaims(current.Claims.Obo)
		if err != nil {
			return err
		}

		current = &Signature{
			Token:  token,
			Parts:  parts,
			Claims: claims,
			JWKS:   s.JWKS,
		}
	}

	return nil
}

func verifySingleJWT(s *Signature, ca *x509.CertPool) error {
	logrus.Infof("Validating JWT: %s, len(JWKS.Keys) = %d", s.Claims.Subject, len(s.JWKS.Keys))

	if len(s.Parts) != 3 {
		return errors.New("length of parts array is incorrect")
	}

	jwk := s.JWKS.Key(s.Claims.Subject)
	if len(jwk) == 0 {
		return errors.Errorf("no JWK with keyID = %s, found in JWKS", s.Claims.Subject)
	}

	// JWKS might contain more than one JWK for specified SpiffeID
	logrus.Infof("%d JWK for %s keyID", len(jwk), s.Claims.Subject)
	for i := 0; i < len(jwk); i++ {
		leaf := jwk[i].Certificates[0]

		// we iterate over all JWK with provided SpiffeID and try to verify JWT
		if err := s.Token.Method.Verify(strings.Join(s.Parts[0:2], "."), s.Parts[2], leaf.PublicKey); err != nil {
			logrus.Info("Wrong JWK, trying next one...")
			continue
		}

		// if we manage to find appropriate JWK, we will check it with our CA
		if err := verifyJWK(s.GetSpiffeID(), &jwk[i], ca); err != nil {
			continue
		}

		return nil
	}

	return errors.New("no appropriate JWK found in JWKS")
}

// verifyJWK verifies that JWK was issued by trusted authority
func verifyJWK(spiffeID string, jwk *jose.JSONWebKey, caBundle *x509.CertPool) error {
	leaf := jwk.Certificates[0]

	if leaf.URIs[0].String() != spiffeID {
		return errors.New("spiffeID provided with JWT not equal to spiffeID from x509 TLS certificate")
	}

	interm := x509.NewCertPool()
	for i, c := range jwk.Certificates {
		if i == 0 {
			continue
		}
		interm.AddCert(c)
	}

	_, err := leaf.Verify(x509.VerifyOptions{
		Roots:         caBundle,
		Intermediates: interm,
	})

	if err != nil {
		return errors.Wrap(err, "certificate is signed by untrusted authority: %s")
	}

	return nil
}
