package tests

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
)

func TestEmptyConnectionContext(t *testing.T) {
	g := NewWithT(t)

	ctx := &connectioncontext.ConnectionContext{}
	g.Expect(ctx.IsValid()).To(BeNil())
}

func TestPrefixConnectionContext(t *testing.T) {
	g := NewWithT(t)

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			SrcRoutes: []*connectioncontext.Route{
				{
					Prefix: "",
				},
			},
		},
	}
	g.Expect(ctx.IsValid().Error()).To(Equal("ConnectionContext.Route.Prefix is required and cannot be empty/nil: src_routes:<> "))

	ctx = &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			DstRoutes: []*connectioncontext.Route{
				{
					Prefix: "",
				},
			},
		},
	}
	g.Expect(ctx.IsValid().Error()).To(Equal("ConnectionContext.Route.Prefix is required and cannot be empty/nil: dst_routes:<> "))
}
func TestPrefixWrongConnectionContext(t *testing.T) {
	g := NewWithT(t)

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			SrcRoutes: []*connectioncontext.Route{
				{
					Prefix: "8.8.8.8",
				},
			},
		},
	}
	g.Expect(ctx.IsValid().Error()).To(Equal("ConnectionContext.Route.Prefix should be a valid CIDR address: src_routes:<prefix:\"8.8.8.8\" > "))

	ctx = &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			DstRoutes: []*connectioncontext.Route{
				{
					Prefix: "8.8.8.8",
				},
			},
		},
	}
	g.Expect(ctx.IsValid().Error()).To(Equal("ConnectionContext.Route.Prefix should be a valid CIDR address: dst_routes:<prefix:\"8.8.8.8\" > "))
}
func TestPrefixFineConnectionContext(t *testing.T) {
	g := NewWithT(t)

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			SrcRoutes: []*connectioncontext.Route{
				{
					Prefix: "8.8.8.8/30",
				},
			},
		},
	}
	g.Expect(ctx.IsValid()).To(BeNil())

	ctx = &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			DstRoutes: []*connectioncontext.Route{
				{
					Prefix: "8.8.8.8/30",
				},
			},
		},
	}
	g.Expect(ctx.IsValid()).To(BeNil())
}

func TestIpNeighbors(t *testing.T) {
	g := NewWithT(t)

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			IpNeighbors: []*connectioncontext.IpNeighbor{
				{
					Ip: "",
				},
			},
		},
	}
	g.Expect(ctx.IsValid().Error()).To(Equal("ConnectionContext.IpNeighbors.Ip is required and cannot be empty/nil: ip_neighbors:<> "))
}

func TestHWNeighbors(t *testing.T) {
	g := NewWithT(t)

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			IpNeighbors: []*connectioncontext.IpNeighbor{
				{
					Ip: "8.8.8.8",
				},
			},
		},
	}
	err := ctx.IsValid()
	g.Expect(err).ShouldNot(BeNil())
	g.Expect(err.Error()).To(Equal("ConnectionContext.IpNeighbors.HardwareAddress is required and cannot be empty/nil: ip_neighbors:<ip:\"8.8.8.8\" > "))
}

func TestValidNeighbors(t *testing.T) {
	g := NewWithT(t)

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			IpNeighbors: []*connectioncontext.IpNeighbor{
				{
					Ip:              "8.8.8.8",
					HardwareAddress: "00:25:96:FF:FE:12:34:56",
				},
			},
		},
	}
	g.Expect(ctx.IsValid()).To(BeNil())
}
