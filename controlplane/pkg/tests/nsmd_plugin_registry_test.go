package tests

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
)

type dummyConnectionPlugin struct {
	prefixes   []string
	shouldFail bool
}

func (cp *dummyConnectionPlugin) UpdateConnectionContext(ctx context.Context, connCtx *connectioncontext.ConnectionContext, opts ...grpc.CallOption) (*connectioncontext.ConnectionContext, error) {
	connCtx.GetIpContext().ExcludedPrefixes = append(connCtx.GetIpContext().GetExcludedPrefixes(), cp.prefixes...)
	return connCtx, nil
}

func (cp *dummyConnectionPlugin) ValidateConnectionContext(ctx context.Context, connCtx *connectioncontext.ConnectionContext, opts ...grpc.CallOption) (*plugins.ConnectionValidationResult, error) {
	if cp.shouldFail {
		return &plugins.ConnectionValidationResult{
			Status:       plugins.ConnectionValidationStatus_FAIL,
			ErrorMessage: "validation failed",
		}, nil
	}

	return &plugins.ConnectionValidationResult{
		Status: plugins.ConnectionValidationStatus_SUCCESS,
	}, nil
}

func TestDummyConnectionPlugin(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	plugin := &dummyConnectionPlugin{
		prefixes: []string{"10.10.1.0/24"},
	}
	srv.pluginRegistry.connectionPluginManager.addPlugin(plugin)

	nsmClient, conn := srv.requestNSMConnection("nsm")
	defer func() { _ = conn.Close() }()

	request := createRequest()

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	originl, ok := srv.serviceRegistry.localTestNSE.(*localTestNSENetworkServiceClient)
	Expect(ok).To(Equal(true))
	Expect(originl.req.Connection.GetContext().GetIpContext().GetExcludedPrefixes()).To(Equal([]string{"10.10.1.0/24"}))
}

func TestDummyConnectionPlugin2(t *testing.T) {
	RegisterTestingT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	defer srv.Stop()
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
	srv.testModel.AddEndpoint(srv.registerFakeEndpoint("golden_network", "test", Master))

	plugin := &dummyConnectionPlugin{
		shouldFail: true,
	}
	srv.pluginRegistry.connectionPluginManager.addPlugin(plugin)

	nsmClient, conn := srv.requestNSMConnection("nsm")
	defer func() { _ = conn.Close() }()

	request := createRequest()

	_, err := nsmClient.Request(context.Background(), request)
	Expect(err).NotTo(BeNil())
	Expect(err.Error()).To(ContainSubstring("failure Validating NSE Connection: validation failed"))

	originl, ok := srv.serviceRegistry.localTestNSE.(*localTestNSENetworkServiceClient)
	Expect(ok).To(Equal(true))
	Expect(originl.req.Connection.GetContext().GetIpContext().GetExcludedPrefixes()).To(BeNil())
}
