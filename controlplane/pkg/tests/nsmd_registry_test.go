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
	RegisterTestingT(t)

	storage := newSharedStorage()

	srv := newNSMDFullServer("nsm1", storage)
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")

	reply, conn := srv.requestNSM("nsm-1")
	defer conn.Close()

	configuration := &common.NSConfiguration{
		Workspace:        reply.Workspace,
		NsmServerSocket:  reply.ClientBaseDir + reply.Workspace + "/" + reply.NsmServerSocket,
		NsmClientSocket:  reply.ClientBaseDir + reply.Workspace + "/" + reply.NsmClientSocket,
		AdvertiseNseName: "test_nse",
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
	Expect(len(endpoints1)).To(Equal(1))
	srv.StopNoClean()

	// We need to restart server
	storage2 := newSharedStorage()
	srv = newNSMDFullServerAt("nsm2", storage2, srv.rootDir)
	srv.addFakeDataplane("test_data_plane", "tcp:some_addr")
	endpoints2 := srv.testModel.GetEndpointsByNetworkService("test_nse")

	Expect(len(endpoints2)).To(Equal(1))
	Expect(endpoints1[0].SocketLocation).To(Equal(endpoints2[0].SocketLocation))
	Expect(endpoints1[0].Workspace).To(Equal(endpoints2[0].Workspace))
	Expect(endpoints1[0].Endpoint.NetworkServiceManager.Name).ToNot(Equal(endpoints2[0].Endpoint.NetworkServiceManager.Name))
}
