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
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"hash/fnv"
	"reflect"
	"strings"
	"time"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"gopkg.in/square/go-jose.v2"

	"github.com/pkg/errors"

	"github.com/dgrijalva/jwt-go"
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

func hash(s string) (uint32, error) {
	h := fnv.New32a()
	if _, err := h.Write([]byte(s)); err != nil {
		return 0, nil
	}
	return h.Sum32(), nil
}

func GenerateSignature(ctx context.Context, msg interface{}, claimsSetter ClaimsSetter, p Provider, opts ...SignOption) (string, error) {
	span := spanhelper.FromContext(ctx, "security.GenerateSignature")
	defer span.Finish()

	cfg := &signConfig{}
	for _, o := range opts {
		o.apply(cfg)
	}

	claims := &ChainClaims{}
	if err := claimsSetter(claims, msg); err != nil {
		return "", err
	}

	if cfg.obo != nil && cfg.obo.GetSpiffeID() == p.GetSpiffeID() {
		span.Logger().Info("GeneratingSignature: claims.Obo.Subject equals current SpiffeID")
		return cfg.obo.ToString()
	}

	if cfg.obo != nil && cfg.obo.Token != nil {
		span.Logger().Info("GeneratingSignature: claims.Obo is not empty")
		claims.Obo = cfg.obo.Token.Raw
	}

	if cfg.expiresAt != 0 {
		claims.ExpiresAt = time.Now().Add(cfg.expiresAt).Unix()
	}

	var xcerts []*x509.Certificate
	span.Logger().Infof("Private key type = %v", reflect.TypeOf(p.GetCertificate().PrivateKey))

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

	h, _ := hash(base64.StdEncoding.EncodeToString(xcerts[0].Raw))
	key := fmt.Sprintf("%s %v", p.GetSpiffeID(), h)

	if len(jwks.Key(key)) == 0 {
		jwks.Keys = append(jwks.Keys, jose.JSONWebKey{
			KeyID:        key,
			Key:          xcerts[0].PublicKey,
			Certificates: xcerts,
		})
	}

	claims.Subject = p.GetSpiffeID()

	token, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(p.GetCertificate().PrivateKey)
	if err != nil {
		return "", err
	}

	return SignatureString(token, jwks)
}

func VerifySignature(ctx context.Context, signature string, ca *x509.CertPool, spiffeID string) error {
	span := spanhelper.FromContext(ctx, "security.VerifySignature")
	defer span.Finish()

	s := &Signature{}
	err := s.Parse(signature)
	if err != nil {
		return err
	}

	return verifySignatureParsed(ctx, s, ca, spiffeID)
}

func verifySignatureParsed(ctx context.Context, s *Signature, ca *x509.CertPool, spiffeID string) error {
	span := spanhelper.FromContext(ctx, "security.verifySignatureParsed")
	defer span.Finish()

	if s.Claims.Subject != spiffeID {
		return errors.New("wrong spiffeID")
	}

	if s.Claims.ExpiresAt != 0 {
		if time.Now().After(time.Unix(s.Claims.ExpiresAt, 0)) {
			return errors.New("token expired")
		}
	}

	return verifyChainJWT(ctx, s, ca)
}

func verifyChainJWT(ctx context.Context, s *Signature, ca *x509.CertPool) error {
	span := spanhelper.FromContext(ctx, "security.verifyChainJWT")
	defer span.Finish()

	current := s
	idxMap := map[string]int{}

	for current != nil {
		err := verifySingleJWT(ctx, current, ca, idxMap[current.GetSpiffeID()])
		if err != nil {
			return err
		}
		idxMap[current.GetSpiffeID()]++

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

func verifySingleJWT(ctx context.Context, s *Signature, ca *x509.CertPool, idx int) error {
	span := spanhelper.FromContext(ctx, "security.verifySingleJWT")
	defer span.Finish()

	span.Logger().Infof("Validating JWT: %s, len(JWKS.Keys) = %d", s.Claims.Subject, len(s.JWKS.Keys))

	if len(s.Parts) != 3 {
		return errors.New("length of parts array is incorrect")
	}

	var jwk []jose.JSONWebKey
	ts := time.Now()
	for _, key := range s.JWKS.Keys {
		if strings.Split(key.KeyID, " ")[0] == s.Claims.Subject {
			jwk = append(jwk, key)
		}
	}
	span.Logger().Infof("jwk search by key takes %v", time.Since(ts))

	if len(jwk) == 0 {
		return errors.Errorf("no JWK with keyID = %s, found in JWKS", s.Claims.Subject)
	}

	offset := idx % len(jwk)
	// JWKS might contain more than one JWK for specified SpiffeID
	span.Logger().Infof("%d JWK for %s keyID, offset %d", len(jwk), s.Claims.Subject, offset)

	for i := 0; i < len(jwk); i++ {
		k := len(jwk) - 1 - (offset+i)%len(jwk)
		leaf := jwk[k].Certificates[0]

		// we iterate over all JWK with provided SpiffeID and try to verify JWT
		tv := time.Now()
		if err := s.Token.Method.Verify(strings.Join(s.Parts[0:2], "."), s.Parts[2], leaf.PublicKey); err != nil {
			span.Logger().Info("Wrong JWK, trying next one...")
			continue
		}
		span.Logger().Infof("s.Token.Verify takes %v", time.Since(tv))

		// if we manage to find appropriate JWK, we will check it with our CA
		if err := verifyJWK(ctx, s.GetSpiffeID(), &jwk[k], ca); err != nil {
			continue
		}

		return nil
	}

	return errors.New("no appropriate JWK found in JWKS")
}

// verifyJWK verifies that JWK was issued by trusted authority
func verifyJWK(ctx context.Context, spiffeID string, jwk *jose.JSONWebKey, caBundle *x509.CertPool) error {
	span := spanhelper.FromContext(ctx, "security.verifyJWK")
	defer span.Finish()

	leaf := jwk.Certificates[0]

	if leaf.URIs[0].String() != spiffeID {
		return errors.New("spiffeID provided with JWT not equal to spiffeID from x509 TLS certificate")
	}

	tp := time.Now()
	interm := x509.NewCertPool()
	for i, c := range jwk.Certificates {
		if i == 0 {
			continue
		}
		interm.AddCert(c)
	}
	span.Logger().Infof("adding certs to pool takes %v", time.Since(tp))

	tv := time.Now()
	_, err := leaf.Verify(x509.VerifyOptions{
		Roots:         caBundle,
		Intermediates: interm,
	})

	span.Logger().Infof("len(caBundle.Subjects()) = %v", len(caBundle.Subjects()))
	span.Logger().Infof("len(jwk.Certificates) = %v", len(jwk.Certificates))
	span.Logger().Infof("leaf.Verify takes %v", time.Since(tv))

	if err != nil {
		return errors.Wrap(err, "certificate is signed by untrusted authority: %s")
	}

	return nil
}
