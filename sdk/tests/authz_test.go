// Copyright (c) 2019 Cisco and/or its affiliates.
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
	"testing"

	"github.com/onsi/gomega"
	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

func TestAuthzEndpoint_Positive(t *testing.T) {
	g := gomega.NewWithT(t)

	policy := `
		package test
	
		default allow = false
	
		allow {
			input.connection.id = "allowed"
		}
	`

	p, err := rego.New(
		rego.Query("data.test.allow"),
		rego.Module("example.com", policy)).PrepareForEval(context.Background())
	g.Expect(err).To(gomega.BeNil())

	srv := endpoint.NewAuthzEndpoint(p)

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id: "allowed",
		},
	}

	resp, err := srv.Request(context.Background(), request)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(resp.Id).To(gomega.Equal("allowed"))
}

func TestAuthzEndpoint_Negative(t *testing.T) {
	g := gomega.NewWithT(t)

	policy := `
		package test

		default allow = false

		allow {
			input.connection.id = "allowed"
		}
	`

	p, err := rego.New(
		rego.Query("data.test.allow"),
		rego.Module("example.com", policy)).PrepareForEval(context.Background())

	g.Expect(err).To(gomega.BeNil())

	srv := endpoint.NewAuthzEndpoint(p)

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id: "not_allowed",
		},
	}

	resp, err := srv.Request(context.Background(), request)
	g.Expect(resp).To(gomega.BeNil())

	s, ok := status.FromError(err)
	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(s.Code()).To(gomega.Equal(codes.PermissionDenied))
}
