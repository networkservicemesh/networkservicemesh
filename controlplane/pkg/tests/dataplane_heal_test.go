package tests

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
	connection2 "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
)

func TestHealLocalDataplane(t *testing.T) {
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	srv2 := NewNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()

	srv.TestModel.AddDataplane(context.Background(),testDataplane1)
	srv2.TestModel.AddDataplane(context.Background(),testDataplane2)

	// Register in both
	nseReg := srv2.registerFakeEndpointWithName("golden_network", "test", Worker, "ep1")

	// Add to local endpoints for Server2
	srv2.TestModel.AddEndpoint(context.Background(),nseReg)

	l1 := newTestConnectionModelListener(Master)
	l2 := newTestConnectionModelListener(Worker)

	srv.TestModel.AddListener(l1)
	srv2.TestModel.AddListener(l2)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					DstIpRequired: true,
					SrcIpRequired: true,
				},
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*connection.Mechanism{
			{
				Type: connection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					connection.NetNsInodeKey:    "10",
					connection.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for cross connections.
	clientConnection1 := srv.TestModel.GetClientConnection(nsmResponse.GetId())
	g.Expect(clientConnection1.GetID()).To(Equal("1"))
	g.Expect(clientConnection1.Xcon.GetRemoteDestination().GetMechanism().GetParameters()[connection2.VXLANSrcIP]).To(Equal("127.0.0.1"))

	clientConnection2 := srv2.TestModel.GetClientConnection(clientConnection1.Xcon.GetRemoteDestination().GetId())
	g.Expect(clientConnection2.GetID()).To(Equal("1"))

	timeout := time.Second * 10

	l1.WaitAdd(1, timeout, t)
	// We need to inform cross connection monitor about this connection, since dataplane is fake one.
	epName := clientConnection1.Endpoint.GetNetworkServiceEndpoint().GetName()
	_, err = srv.nseRegistry.RemoveNSE(context.Background(), &registry.RemoveNSERequest{
		NetworkServiceEndpointName: epName,
	})
	if err != nil {
		t.Fatal("Err must be nil")
	}

	// Simulate dataplane dead
	srv.TestModel.AddDataplane(context.Background(), testDataplane1_1)
	srv.TestModel.DeleteDataplane(context.Background(),testDataplane1.RegisteredName)

	//<- time.After(1000*time.Second)
	// We need to inform cross connection monitor about this connection, since dataplane is fake one.
	// First update is with down state
	// But we want to wait for Up state
	l1.WaitUpdate(5, timeout, t)
	// We need to inform cross connection monitor about this connection, since dataplane is fake one.

	clientConnection1_1 := srv.TestModel.GetClientConnection(nsmResponse.GetId())
	g.Expect(clientConnection1_1 != nil).To(Equal(true))
	g.Expect(clientConnection1_1.GetID()).To(Equal("1"))
	g.Expect(clientConnection1_1.Xcon.GetRemoteDestination().GetId()).To(Equal("1"))
	g.Expect(clientConnection1_1.Xcon.GetRemoteDestination().GetNetworkServiceEndpointName()).To(Equal(epName))
	g.Expect(clientConnection1_1.Xcon.GetRemoteDestination().GetMechanism().GetParameters()[connection2.VXLANSrcIP]).To(Equal("127.0.0.7"))
}
