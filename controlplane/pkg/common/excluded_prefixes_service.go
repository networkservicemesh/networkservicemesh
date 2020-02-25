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
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/sdk/prefix_pool"
)

// ConnectionService makes basic Mechanism selection for the incoming connection
type excludedPrefixesService struct {
	prefixes prefix_pool.PrefixPool
}

// NewExcludedPrefixesService -  creates a service to select endpoint.
func NewExcludedPrefixesService() networkservice.NetworkServiceServer {
	return NewExcludedPrefixesServiceFromPath(prefix_pool.PrefixesFilePathDefault)
}

// NewExcludedPrefixesService -  creates a service to select endpoint.
func NewExcludedPrefixesServiceFromPath(configPath string) networkservice.NetworkServiceServer {
	return &excludedPrefixesService{
		prefixes: prefix_pool.NewPrefixPoolReader(configPath),
	}
}

func (eps *excludedPrefixesService) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logger := Log(ctx)

	if request.GetConnection() == nil {
		return nil, errors.Errorf("request's connection cannot be empty")
	}

	requestNext := request.Clone()
	conn := requestNext.Connection
	if conn.Context == nil {
		conn.Context = &networkservice.ConnectionContext{}
	}
	if conn.Context.IpContext == nil {
		conn.Context.IpContext = &networkservice.IPContext{}
	}
	prefixes := eps.prefixes.GetPrefixes()
	logger.Infof("ExcludedPrefixesService: adding excluded prefixes to connection: %v", prefixes)
	ipCtx := conn.Context.IpContext
	ipCtx.ExcludedPrefixes = append(ipCtx.GetExcludedPrefixes(), prefixes...)

	conn, err := ProcessNext(ctx, requestNext)
	if err != nil {
		return nil, err
	}

	if err = eps.validateConnection(conn); err != nil {
		logger.Errorf("ExcludedPrefixesService: connection is invalid: %v", err)
		return nil, err
	}

	return conn, nil
}

func (eps *excludedPrefixesService) validateConnection(conn *networkservice.Connection) error {
	if err := conn.IsComplete(); err != nil {
		return err
	}

	ipCtx := conn.GetContext().GetIpContext()
	if err := eps.validateIPAddress(ipCtx.GetSrcIpAddr(), "srcIP"); err != nil {
		return err
	}

	return eps.validateIPAddress(ipCtx.GetDstIpAddr(), "dstIP")
}

func (eps *excludedPrefixesService) validateIPAddress(ip, ipName string) error {
	if ip == "" {
		return nil
	}
	intersect, err := eps.prefixes.Intersect(ip)
	if err != nil {
		return err
	}
	if intersect {
		return errors.Errorf("%s '%s' intersects excluded prefixes list %v", ipName, ip, eps.prefixes.GetPrefixes())
	}
	return nil
}

func (eps *excludedPrefixesService) Close(ctx context.Context, connection *networkservice.Connection) (*empty.Empty, error) {
	return ProcessClose(ctx, connection)
}
