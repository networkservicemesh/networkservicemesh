// Copyright 2018, 2019 VMware, Inc.
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
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/sirupsen/logrus"
)

type IpamEndpoint struct {
	BaseCompositeEndpoint
	prefixPool prefix_pool.PrefixPool
}

// Request implements the request handler
func (ice *IpamEndpoint) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {

	if ice.GetNext() == nil {
		err := fmt.Errorf("IPAM needs next")
		logrus.Errorf("%v", err)
		return nil, err
	}

	newConnection, err := ice.GetNext().Request(ctx, request)
	if err != nil {
		logrus.Errorf("Next request failed: %v", err)
		return nil, err
	}

	/* Exclude the prefixes from the pool of available prefixes */
	excludedPrefixes, err := ice.prefixPool.ExcludePrefixes(request.Connection.Context.ExcludedPrefixes)
	if err != nil {
		return nil, err
	}

	//TODO: We need to somehow support IPv6.
	srcIP, dstIP, prefixes, err := ice.prefixPool.Extract(request.Connection.Id, connectioncontext.IpFamily_IPV4, request.Connection.Context.ExtraPrefixRequest...)
	if err != nil {
		return nil, err
	}

	/* Release the actual prefixes that were excluded during IPAM */
	err = ice.prefixPool.ReleaseExcludedPrefixes(excludedPrefixes)
	if err != nil {
		return nil, err
	}

	// Update source/dst IP's
	newConnection.Context.SrcIpAddr = srcIP.String()
	newConnection.Context.DstIpAddr = dstIP.String()

	//Add extra routes.
	newConnection.Context.Routes = []*connectioncontext.Route{
		&connectioncontext.Route{
			Prefix: "8.8.8.8/30",
		},
	}
	newConnection.Context.ExtraPrefixes = prefixes

	addrs, err := net.Interfaces()
	if err == nil {
		for _, iface := range addrs {
			adrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, a := range adrs {
				addr, _, _ := net.ParseCIDR(a.String())
				if !addr.IsLoopback() {
					newConnection.Context.IpNeighbors = append(newConnection.Context.IpNeighbors,
						&connectioncontext.IpNeighbor{
							Ip:              addr.String(),
							HardwareAddress: iface.HardwareAddr.String(),
						},
					)
				}
			}
		}
	}

	err = newConnection.IsComplete()
	if err != nil {
		logrus.Errorf("New connection is not complete: %v", err)
		return nil, err
	}

	logrus.Infof("IPAM completed on connection: %v", newConnection)
	return newConnection, nil
}

// Close implements the close handler
func (ice *IpamEndpoint) Close(ctx context.Context, connection *connection.Connection) (*empty.Empty, error) {
	prefix, requests, err := ice.prefixPool.GetConnectionInformation(connection.GetId())
	logrus.Infof("Release connection prefixes network: %s extra requests: %v", prefix, requests)
	if err != nil {
		logrus.Errorf("Error: %v", err)
	}
	err = ice.prefixPool.Release(connection.GetId())
	if ice.GetNext() != nil {
		return ice.GetNext().Close(ctx, connection)
	}
	return &empty.Empty{}, nil
}

// NewIpamEndpoint creates a IpamEndpoint
func NewIpamEndpoint(configuration *common.NSConfiguration) *IpamEndpoint {
	// ensure the env variables are processed
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}
	configuration.CompleteNSConfiguration()

	pool, err := prefix_pool.NewPrefixPool(configuration.IPAddress)
	if err != nil {
		panic(err.Error())
	}

	rand.Seed(time.Now().UTC().UnixNano())

	self := &IpamEndpoint{
		prefixPool: pool,
	}

	return self
}
