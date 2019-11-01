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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"gopkg.in/square/go-jose.v2"
	"strings"
)

type Context interface {
	GetRequestOboToken() *Signature
	SetRequestOboToken(token *Signature)

	GetResponseOboToken() *Signature
	SetResponseOboToken(token *Signature)
}

type contextImpl struct {
	requestOboSignature  *Signature
	responseOboSignature *Signature
}

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

func NewContext() Context {
	return &contextImpl{}
}

func (c *contextImpl) GetRequestOboToken() *Signature {
	return c.requestOboSignature
}

func (c *contextImpl) SetRequestOboToken(token *Signature) {
	c.requestOboSignature = token
}

func (c *contextImpl) GetResponseOboToken() *Signature {
	return c.responseOboSignature
}

func (c *contextImpl) SetResponseOboToken(token *Signature) {
	c.responseOboSignature = token
}

func WithSecurityContext(parent context.Context, sc Context) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, securityContextKey, sc)
}

func SecurityContext(ctx context.Context) Context {
	value := ctx.Value(securityContextKey)
	if value == nil {
		return nil
	}
	return value.(Context)
}
