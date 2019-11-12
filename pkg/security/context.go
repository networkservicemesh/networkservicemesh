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
