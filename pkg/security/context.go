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

import "context"

type Context interface {
	GetRequestOboToken() string
	SetRequestOboToken(token string)

	GetResponseOboToken() string
	SetResponseOboToken(token string)

	GetClaims() *ChainClaims
	SetClaims(claims *ChainClaims)
}

type contextImpl struct {
	requestOboToken  string
	responseOboToken string
	claims           *ChainClaims
}

func NewContext() Context {
	return &contextImpl{}
}

func (c *contextImpl) GetRequestOboToken() string {
	return c.requestOboToken
}

func (c *contextImpl) SetRequestOboToken(token string) {
	c.requestOboToken = token
}

func (c *contextImpl) GetResponseOboToken() string {
	return c.responseOboToken
}

func (c *contextImpl) SetResponseOboToken(token string) {
	c.responseOboToken = token
}

func (c *contextImpl) GetClaims() *ChainClaims {
	return c.claims
}

func (c *contextImpl) SetClaims(claims *ChainClaims) {
	c.claims = claims
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
