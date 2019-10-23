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
	"encoding/base64"
	"time"

	"github.com/pkg/errors"

	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"
)

type Signable interface {
	SetSignature(sign string)
}

type Signed interface {
	GetSignature() string
}

type ClaimsSetter func(claims *ChainClaims, msg interface{}) error

type SignOption interface {
	apply(*signConfig)
}

type signConfig struct {
	obo       *TokenAndClaims
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

func WithObo(obo *TokenAndClaims) SignOption {
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

	if cfg.obo != nil && cfg.obo.claims != nil && cfg.obo.claims.Subject == p.GetSpiffeID() {
		logrus.Info("GeneratingSignature: claims.Obo.Subject equals current SpiffeID")
		return cfg.obo.Token, nil
	}

	if cfg.obo != nil && cfg.obo.Token != "" {
		logrus.Info("GeneratingSignature: claims.Obo is not empty")
		claims.Obo = cfg.obo.Token
	}

	if cfg.expiresAt != 0 {
		claims.ExpiresAt = time.Now().Add(cfg.expiresAt).Unix()
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

	return verifySignatureParsed(token, parts, claims, ca, spiffeID)
}

func verifySignatureParsed(token *jwt.Token, parts []string, claims *ChainClaims, ca *x509.CertPool, spiffeID string) error {
	if claims.Subject != spiffeID {
		return errors.New("wrong spiffeID")
	}

	if claims.ExpiresAt != 0 {
		if time.Now().After(time.Unix(claims.ExpiresAt, 0)) {
			return errors.New("token expired")
		}
	}

	return verifyJWTChain(token, parts, claims, ca)
}
