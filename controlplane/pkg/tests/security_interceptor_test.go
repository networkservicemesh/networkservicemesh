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

package tests

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/security"
	testsec "github.com/networkservicemesh/networkservicemesh/pkg/security/test"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

func testNewServerFunc(provider security.Provider) tools.NewServerFunc {
	cfg := &tools.DialConfig{SecurityProvider: provider, OpenTracing: true}
	return new(tools.NewServerBuilder).TokenVerification(&common.NSTokenConfig{}).WithConfig(cfg).NewServerFunc()
}

func testDialFunc(provider security.Provider) tools.DialContextFunc {
	cfg := &tools.DialConfig{SecurityProvider: provider, OpenTracing: true}
	return new(tools.DialBuilder).TCP().TokenVerification(&common.NSTokenConfig{}).WithConfig(cfg).DialContextFunc()
}

type dummyNetworkService struct {
	dialContextFunc tools.DialContextFunc
	newServerFunc   tools.NewServerFunc
	provider        security.Provider
	transitions     map[string]map[string]string
	ipaddrs         map[string]string
	name            string
}

func newDummyNetworkService(name string, provider security.Provider,
	t map[string]map[string]string, ipaddrs map[string]string) *dummyNetworkService {
	return &dummyNetworkService{
		newServerFunc:   testNewServerFunc(provider),
		dialContextFunc: testDialFunc(provider),
		provider:        provider,
		transitions:     t,
		name:            name,
		ipaddrs:         ipaddrs,
	}
}

func (d *dummyNetworkService) start(address string) (closeFunc func(), err error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	srv := d.newServerFunc(context.Background())
	networkservice.RegisterNetworkServiceServer(srv, d)

	go func() {
		if err := srv.Serve(ln); err != nil {
			logrus.Error(err)
		}
	}()

	return func() { srv.Stop() }, nil
}

func (d *dummyNetworkService) Request(ctx context.Context, r *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	logrus.Infof("Request on %s, from - %s", d.name, r.Connection.Id)
	next, ok := d.transitions[d.name][r.Connection.Id]
	if !ok {
		rv := &connection.Connection{Id: "1"}
		sign, err := security.GenerateSignature(rv, common.ConnectionFillClaimsFunc, d.provider)
		if err != nil {
			return nil, err
		}
		rv.ResponseJWT = sign
		return rv, nil
	}

	nextAddr := d.ipaddrs[next]
	logrus.Infof("next - %s, next ip - %s", next, nextAddr)

	dialContextFunc := testDialFunc(d.provider)
	conn, err := dialContextFunc(context.Background(), nextAddr)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	nsclient := networkservice.NewNetworkServiceClient(conn)

	reply, err := nsclient.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id: d.name, // Connection.Id is treated like "from"
		},
	})

	//logrus.Info(security.SecurityContext(ctx).GetRequestOboToken())
	//logrus.Info(security.SecurityContext(ctx).GetResponseOboToken())

	sign, err := security.GenerateSignature(reply, common.ConnectionFillClaimsFunc, d.provider,
		security.WithObo(security.SecurityContext(ctx).GetResponseOboToken()))
	if err != nil {
		return nil, err
	}
	reply.ResponseJWT = sign

	return reply, nil

	//return rv, nil
}

func (d *dummyNetworkService) Close(context.Context, *connection.Connection) (*empty.Empty, error) {
	panic("implement me")
}

func newTestSecurityProvider(ca *tls.Certificate, spiffeID string) security.Provider {
	obt := testsec.NewTestCertificateObtainerWithCA(spiffeID, ca, 1*time.Second)
	return security.NewProviderWithCertObtainer(obt)
}

func TestSecurityInterceptor_Chain(t *testing.T) {
	g := NewWithT(t)

	ca, err := testsec.GenerateCA()
	g.Expect(err).To(BeNil())

	ipaddrs := map[string]string{
		"nsmgr-master": "localhost:5252",
		"nsmgr-worker": "localhost:5353",
		"firewall":     "localhost:5450",
		"ps-1":         "localhost:5451",
		"ps-2":         "localhost:5452",
		"ps-3":         "localhost:5453",
		"ps-4":         "localhost:5454",
		"ps-5":         "localhost:5455",
		"vpn-gateway":  "localhost:5456",
	}

	ids := map[string]string{
		"nsmgr-master": "nsmgr",
		"nsmgr-worker": "nsmgr",
		"firewall":     "nse",
		"ps-1":         "nse",
		"ps-2":         "nse",
		"ps-3":         "nse",
		"ps-4":         "nse",
		"ps-5":         "nse",
		"vpn-gateway":  "nse",
	}

	transitions := map[string]map[string]string{
		"nsmgr-master": {
			"nsc":          "nsmgr-worker",
			"nsmgr-worker": "ps-1",
			"ps-1":         "ps-2",
			"ps-2":         "ps-3",
			"ps-3":         "ps-4",
			"ps-4":         "ps-5",
			"ps-5":         "vpn-gateway",
		},
		"nsmgr-worker": {
			"nsmgr-master": "firewall",
			"firewall":     "nsmgr-master",
		},
		"firewall": {
			"nsmgr-worker": "nsmgr-worker",
		},
		"ps-1": {
			"nsmgr-master": "nsmgr-master",
		},
		"ps-2": {
			"nsmgr-master": "nsmgr-master",
		},
		"ps-3": {
			"nsmgr-master": "nsmgr-master",
		},
		"ps-4": {
			"nsmgr-master": "nsmgr-master",
		},
		"ps-5": {
			"nsmgr-master": "nsmgr-master",
		},
	}

	// start all services
	for name, ipaddr := range ipaddrs {
		srv := newDummyNetworkService(
			name, newTestSecurityProvider(&ca, fmt.Sprintf("spiffe://test.com/%s", ids[name])),
			transitions, ipaddrs)
		closeFunc, err := srv.start(ipaddr)
		g.Expect(err).To(BeNil())
		defer closeFunc()
	}

	nscProvider := newTestSecurityProvider(&ca, "spiffe://test.com/nsc")
	dialContextFunc := testDialFunc(nscProvider)
	conn, err := dialContextFunc(context.Background(), "localhost:5252")
	g.Expect(err).To(BeNil())
	defer func() { _ = conn.Close() }()
	nsclient := networkservice.NewNetworkServiceClient(conn)

	reply, err := nsclient.Request(context.Background(), &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id: "nsc", // Connection.Id is treated like "from"
		},
	})
	g.Expect(err).To(BeNil())
	logrus.Info(reply)
}
