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

func TestAuthEndpoint_Positive(t *testing.T) {
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

	srv := endpoint.NewAuthEndpoint(p)

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			Id: "allowed",
		},
	}

	resp, err := srv.Request(context.Background(), request)
	g.Expect(err).To(gomega.BeNil())
	g.Expect(resp.Id).To(gomega.Equal("allowed"))
}

func TestAuthEndpoint_Negative(t *testing.T) {
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

	srv := endpoint.NewAuthEndpoint(p)

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
