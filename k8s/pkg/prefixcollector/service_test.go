package prefixcollector

import (
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/prefix_pool"
	. "github.com/onsi/gomega"
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
	RegisterTestingT(t)

	service, _ := newTestPrefixService("10.10.1.0/24", "10.32.1.0/16")

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{},
	}

	var err error
	ctx, err = service.UpdateConnectionContext(nil, ctx)

	Expect(err).To(BeNil())
	Expect(ctx.GetIpContext().GetExcludedPrefixes()).To(Equal([]string{"10.10.1.0/24", "10.32.1.0/16"}))
}

func TestPrefixServiceValidateConnection(t *testing.T) {
	RegisterTestingT(t)

	service, _ := newTestPrefixService("10.10.1.0/24", "10.32.1.0/16")

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			SrcIpAddr: "10.10.2.0/32",
			DstIpAddr: "10.33.1.0/32",
		},
	}

	result, err := service.ValidateConnectionContext(nil, ctx)

	Expect(err).To(BeNil())
	Expect(result.GetStatus()).To(Equal(plugins.ConnectionValidationStatus_SUCCESS))
	Expect(result.GetErrorMessage()).To(Equal(""))
}

func TestPrefixServiceValidateConnectionFailed(t *testing.T) {
	RegisterTestingT(t)

	service, _ := newTestPrefixService("10.10.1.0/24", "10.32.1.0/16")

	ctx := &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			SrcIpAddr: "10.10.1.0/32",
		},
	}

	result, err := service.ValidateConnectionContext(nil, ctx)

	Expect(err).To(BeNil())
	Expect(result.GetStatus()).To(Equal(plugins.ConnectionValidationStatus_FAIL))
	Expect(result.GetErrorMessage()).To(Equal("srcIP intersects excluded prefixes list"))

	ctx = &connectioncontext.ConnectionContext{
		IpContext: &connectioncontext.IPContext{
			DstIpAddr: "10.32.1.1/32",
		},
	}

	result, err = service.ValidateConnectionContext(nil, ctx)

	Expect(err).To(BeNil())
	Expect(result.GetStatus()).To(Equal(plugins.ConnectionValidationStatus_FAIL))
	Expect(result.GetErrorMessage()).To(Equal("dstIP intersects excluded prefixes list"))
}
