package tests

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
)

type dummyConnectionPlugin struct {
	prefixes   []string
	shouldFail bool
}

func (cp *dummyConnectionPlugin) UpdateConnection(ctx context.Context, wrapper *plugins.ConnectionWrapper) (*plugins.ConnectionWrapper, error) {
	connCtx := wrapper.GetConnection().GetContext()
	connCtx.GetIpContext().ExcludedPrefixes = append(connCtx.GetIpContext().GetExcludedPrefixes(), cp.prefixes...)
	return wrapper, nil
}

func (cp *dummyConnectionPlugin) ValidateConnection(ctx context.Context, wrapper *plugins.ConnectionWrapper) (*plugins.ConnectionValidationResult, error) {
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
	g := NewWithT(t)

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
	g.Expect(err).To(BeNil())
	g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	originl, ok := srv.serviceRegistry.localTestNSE.(*localTestNSENetworkServiceClient)
	g.Expect(ok).To(Equal(true))
	g.Expect(originl.req.Connection.GetContext().GetIpContext().GetExcludedPrefixes()).To(Equal([]string{"10.10.1.0/24"}))
}

func TestDummyConnectionPlugin2(t *testing.T) {
	g := NewWithT(t)

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
	g.Expect(err).NotTo(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("failure Validating NSE Connection: validation failed"))

	originl, ok := srv.serviceRegistry.localTestNSE.(*localTestNSENetworkServiceClient)
	g.Expect(ok).To(Equal(true))
	g.Expect(originl.req.Connection.GetContext().GetIpContext().GetExcludedPrefixes()).To(BeNil())
}
