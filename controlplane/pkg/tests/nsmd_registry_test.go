package tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"github.com/networkservicemesh/networkservicemesh/sdk/endpoint"
)

func TestNSMDRestart1(t *testing.T) {
	g := NewWithT(t)

	storage := newSharedStorage()

	srv := newNSMDFullServer("nsm1", storage)
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	reply := srv.requestNSM("nsm-1")

	configuration := &common.NSConfiguration{
		Workspace:              reply.Workspace,
		NsmServerSocket:        reply.ClientBaseDir + reply.Workspace + "/" + reply.NsmServerSocket,
		NsmClientSocket:        reply.ClientBaseDir + reply.Workspace + "/" + reply.NsmClientSocket,
		EndpointNetworkService: "test_nse",
	}

	composite := endpoint.NewCompositeEndpoint(
		endpoint.NewMonitorEndpoint(configuration),
		endpoint.NewIpamEndpoint(configuration),
		endpoint.NewConnectionEndpoint(configuration),
	)

	nsmEndpoint, err := endpoint.NewNSMEndpoint(nil, configuration, composite)
	if err != nil {
		logrus.Fatalf("%v", err)
	}

	l1 := newTestConnectionModelListener()
	srv.testModel.AddListener(l1)
	_ = nsmEndpoint.Start()
	defer nsmEndpoint.Delete()

	// Wait for at least one NSE is available.
	l1.WaitEndpoints(1, time.Second*30, t)

	endpoints1 := srv.testModel.GetEndpointsByNetworkService("test_nse")
	g.Expect(len(endpoints1)).To(Equal(1))
	srv.StopNoClean()

	// We need to restart server
	storage2 := newSharedStorage()
	srv = newNSMDFullServerAt("nsm2", storage2, srv.rootDir)
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
	endpoints2 := srv.testModel.GetEndpointsByNetworkService("test_nse")

	g.Expect(len(endpoints2)).To(Equal(1))
	g.Expect(endpoints1[0].SocketLocation).To(Equal(endpoints2[0].SocketLocation))
	g.Expect(endpoints1[0].Workspace).To(Equal(endpoints2[0].Workspace))
	g.Expect(endpoints1[0].Endpoint.NetworkServiceManager.Name).ToNot(Equal(endpoints2[0].Endpoint.NetworkServiceManager.Name))
}
