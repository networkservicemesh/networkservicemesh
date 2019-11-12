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
	"github.com/dgrijalva/jwt-go"
)

type contextKey int

const (
	securityContextKey contextKey = iota
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
