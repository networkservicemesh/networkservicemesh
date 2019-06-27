package tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
)

func TestEmptyConnectionContext(t *testing.T) {
	RegisterTestingT(t)

	ctx := &connectioncontext.ConnectionContext{IpContext: &connectioncontext.IPContext{}}
	Expect(ctx.IpContext.Validate()).To(BeNil())
}

func TestPrefixConnectionContext(t *testing.T) {
	RegisterTestingT(t)

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			Routes: []*connectioncontext.Route{
				&connectioncontext.Route{
					Prefix: "",
				},
			},
		},
	}
	Expect(ctx.IpContext.Validate().Error()).To(Equal("connectionContext.Route.Prefix is required and cannot be empty/nil: routes:<> "))
}
func TestPrefixWrongConnectionContext(t *testing.T) {
	RegisterTestingT(t)

	ctx := &connectioncontext.IPContext{
		Routes: []*connectioncontext.Route{
			&connectioncontext.Route{
				Prefix: "8.8.8.8",
			},
		},
	}
	Expect(ctx.Validate().Error()).To(Equal("connectionContext.Route.Prefix should be a valid CIDR address: routes:<prefix:\"8.8.8.8\" > "))
}
func TestPrefixFineConnectionContext(t *testing.T) {
	RegisterTestingT(t)

	ctx := &connectioncontext.IPContext{
		Routes: []*connectioncontext.Route{
			&connectioncontext.Route{
				Prefix: "8.8.8.8/30",
			},
		},
	}
	Expect(ctx.Validate()).To(BeNil())
}

func TestIpNeighbors(t *testing.T) {
	RegisterTestingT(t)

	ctx := &connectioncontext.IPContext{
		IpNeighbors: []*connectioncontext.IpNeighbor{
			&connectioncontext.IpNeighbor{
				Ip: "",
			},
		},
	}
	Expect(ctx.Validate().Error()).To(Equal("connectionContext.IpNeighbors.Ip is required and cannot be empty/nil: ip_neighbors:<> "))
}

func TestHWNeighbors(t *testing.T) {
	RegisterTestingT(t)

	ctx := &connectioncontext.IPContext{
		IpNeighbors: []*connectioncontext.IpNeighbor{
			&connectioncontext.IpNeighbor{
				Ip: "8.8.8.8",
			},
		},
	}
	err := ctx.Validate()
	Expect(err).ShouldNot(BeNil())
	Expect(err.Error()).To(Equal("connectionContext.IpNeighbors.HardwareAddress is required and cannot be empty/nil: ip_neighbors:<ip:\"8.8.8.8\" > "))
}

func TestValidNeighbors(t *testing.T) {
	RegisterTestingT(t)

	ctx := &connectioncontext.IPContext{
		IpNeighbors: []*connectioncontext.IpNeighbor{
			&connectioncontext.IpNeighbor{
				Ip:              "8.8.8.8",
				HardwareAddress: "00:25:96:FF:FE:12:34:56",
			},
		},
	}
	Expect(ctx.Validate()).To(BeNil())
}
