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
package endpoint

import (
	"context"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

type addDnsConfigs struct {
	dnsConfigs []*connectioncontext.DNSConfig
}

func NewAddDNSConfigs(dnsConfigs ...*connectioncontext.DNSConfig) networkservice.NetworkServiceServer {
	return addDnsConfigs{dnsConfigs: dnsConfigs}
}

func (a addDnsConfigs) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	ensureDnsContextPresent(request)
	dnsConfigs := request.GetConnection().GetContext().GetDnsContext().GetConfigs()
	dnsConfigs = append(dnsConfigs, a.dnsConfigs...)
	request.GetConnection().GetContext().GetDnsContext().Configs = dnsConfigs
	if Next(ctx) != nil {
		return Next(ctx).Request(ctx, request)
	}
	return request.GetConnection(), nil
}

func (a addDnsConfigs) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, conn)
	}
	return &empty.Empty{}, nil
}

type addDnsConfigDstIp struct {
	searchDomains []string
}

func NewAddDnsConfigDstIp(searchDomains ...string) networkservice.NetworkServiceServer {
	return &addDnsConfigDstIp{searchDomains: searchDomains}
}

func (a addDnsConfigDstIp) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	conn := request.GetConnection()
	if Next(ctx) != nil {
		var err error
		conn, err = Next(ctx).Request(ctx, request)
		if err != nil {
			return nil, err
		}
	}
	dstIp := conn.GetContext().GetIpContext().GetDstIpAddr()
	if dstIp != "" {
		ensureDnsContextPresent(request)
		dstIp = strings.Split(dstIp, "/")[0]
		dnsConfigs := conn.GetContext().GetDnsContext().GetConfigs()
		dnsConfigs = append(dnsConfigs, &connectioncontext.DNSConfig{
			DnsServerIps:  []string{dstIp},
			SearchDomains: a.searchDomains,
		})
		conn.GetContext().GetDnsContext().Configs = dnsConfigs
	}
	return conn, nil
}

func (a addDnsConfigDstIp) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if Next(ctx) != nil {
		return Next(ctx).Close(ctx, conn)
	}
	return &empty.Empty{}, nil
}

func ensureDnsContextPresent(request *networkservice.NetworkServiceRequest) {
	if request.GetConnection() == nil {
		request.Connection = &connection.Connection{}
	}
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = &connectioncontext.ConnectionContext{}
	}
	if request.GetConnection().GetContext().GetDnsContext() == nil {
		request.GetConnection().GetContext().DnsContext = &connectioncontext.DNSContext{}
	}
}
