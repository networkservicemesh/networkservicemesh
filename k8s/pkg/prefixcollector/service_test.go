package prefixcollector

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"
)

func newTestPrefixService(prefixes ...string) (plugins.ConnectionPluginServer, error) {
	prefixPool, err := prefix_pool.NewPrefixPool(prefixes...)
	if err != nil {
		return nil, err
	}

	return &prefixService{
		excludedPrefixes: prefixPool,
	}, nil
}

func TestPrefixServiceUpdateConnection(t *testing.T) {
	g := NewWithT(t)

	service, err := newTestPrefixService("10.10.1.0/24", "10.32.1.0/16")
	g.Expect(err).To(BeNil())

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{},
	}

	ctx, err = service.UpdateConnectionContext(context.TODO(), ctx)

	g.Expect(err).To(BeNil())
	g.Expect(ctx.GetIpContext().GetExcludedPrefixes()).To(Equal([]string{"10.10.1.0/24", "10.32.1.0/16"}))
}

func TestPrefixServiceValidateConnection(t *testing.T) {
	g := NewWithT(t)

	service, err := newTestPrefixService("10.10.1.0/24", "10.32.1.0/16")
	g.Expect(err).To(BeNil())

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			SrcIpAddr: "10.10.2.0/32",
			DstIpAddr: "10.33.1.0/32",
		},
	}

	var result *plugins.ConnectionValidationResult
	result, err = service.ValidateConnectionContext(context.TODO(), ctx)

	g.Expect(err).To(BeNil())
	g.Expect(result.GetStatus()).To(Equal(plugins.ConnectionValidationStatus_SUCCESS))
	g.Expect(result.GetErrorMessage()).To(Equal(""))
}

func TestPrefixServiceValidateConnectionFailed(t *testing.T) {
	g := NewWithT(t)

	service, err := newTestPrefixService("10.10.1.0/24", "10.32.1.0/16")
	g.Expect(err).To(BeNil())

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			SrcIpAddr: "10.10.1.0/32",
		},
	}

	var result *plugins.ConnectionValidationResult
	result, err = service.ValidateConnectionContext(context.TODO(), ctx)

	g.Expect(err).To(BeNil())
	g.Expect(result.GetStatus()).To(Equal(plugins.ConnectionValidationStatus_FAIL))
	g.Expect(result.GetErrorMessage()).To(Equal("srcIP intersects excluded prefixes list"))

	ctx = &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			DstIpAddr: "10.32.1.1/32",
		},
	}

	result, err = service.ValidateConnectionContext(context.TODO(), ctx)

	g.Expect(err).To(BeNil())
	g.Expect(result.GetStatus()).To(Equal(plugins.ConnectionValidationStatus_FAIL))
	g.Expect(result.GetErrorMessage()).To(Equal("dstIP intersects excluded prefixes list"))
}
