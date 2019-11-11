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
	"os"
	"path"
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/prefix_pool"
)

var (
	prefixes = []string{"172.16.1.0/24", "10.32.0.0/12", "10.96.0.0/12"}
)

// TestExcludePrefixesInjection checks excluded prefixes injection
func TestExcludePrefixesInjection(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	request := buildRequest()
	conn, err := doRequest(g, request)

	g.Expect(err).To(gomega.BeNil())
	connPrefixes := conn.GetContext().GetIpContext().GetExcludedPrefixes()
	g.Expect(connPrefixes).To(gomega.ConsistOf(prefixes))
}

// TestExcludePrefixesValidation checks IP address validation
func TestExcludePrefixesValidation(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	request := buildRequest()

	request.Connection.Context = &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{DstIpAddr: "10.32.0.1/32"},
	}
	_, err := doRequest(g, request)
	g.Expect(err.Error()).To(gomega.MatchRegexp("dstIP .* intersects excluded prefixes list"))

	request.Connection.Context = &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{SrcIpAddr: "10.32.0.1/32"},
	}
	_, err = doRequest(g, request)
	g.Expect(err.Error()).To(gomega.MatchRegexp("srcIP .* intersects excluded prefixes list"))
}

func buildRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id:             "id",
			NetworkService: "foo_service",
		},
	}
}

func doRequest(g *gomega.GomegaWithT, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	// create folder for config
	configDir := os.TempDir()
	err := os.MkdirAll(configDir, os.ModeDir|os.ModePerm)
	g.Expect(err).To(gomega.BeNil())
	defer os.Remove(configDir)

	// write excluded prefixes yaml
	configPath := path.Join(configDir, prefix_pool.PrefixesFile)
	f, err := os.Create(configPath)
	g.Expect(err).To(gomega.BeNil())
	defer os.Remove(configPath)
	f.WriteString(strings.Join(append([]string{"prefixes:"}, prefixes...), "\n- "))
	f.Close()

	// ExcludedPrefixesService is expected to read excluded prefixes we just have written to file
	prefixeService := common.NewExcludedPrefixesServiceFromPath(configPath)

	return prefixeService.Request(context.Background(), request)
}
