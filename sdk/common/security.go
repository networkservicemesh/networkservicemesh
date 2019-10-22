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

package common

import (
	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	connection2 "github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"

	unifiedns "github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/security"
)

type NSTokenConfig struct {
}

func (cfg *NSTokenConfig) FillClaims(claims *security.ChainClaims, msg interface{}) error {
	if request, ok := msg.(networkservice.Request); ok {
		claims.Audience = request.GetRequestConnection().GetNetworkService()
		return nil
	}

	if request, ok := msg.(*unifiedns.NetworkServiceRequest); ok {
		claims.Audience = request.GetRequestConnection().GetNetworkService()
		return nil
	}

	return errors.New("unable to cast msg to networkservice's request")
}

func (cfg *NSTokenConfig) RequestFilter(req interface{}) bool {
	if _, ok := req.(networkservice.Request); ok {
		return true
	}

	if _, ok := req.(*unifiedns.NetworkServiceRequest); ok {
		return true
	}

	return false
}

func ConnectionFillClaimsFunc(claims *security.ChainClaims, msg interface{}) error {
	if conn, ok := msg.(*connection.Connection); ok {
		claims.Audience = conn.GetNetworkService()
		return nil
	}

	if conn, ok := msg.(connection2.Connection); ok {
		claims.Audience = conn.GetNetworkService()
		return nil
	}

	return errors.New("unable to cast msg to connection.Connection")
}
