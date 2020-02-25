package tests

import (
	"os"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/common"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/vxlan"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	. "github.com/onsi/gomega"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

func TestHealLocalForwarder(t *testing.T) {
	_ = os.Setenv(tools.InsecureEnv, "true")
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	srv2 := NewNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()

	srv.TestModel.AddForwarder(context.Background(), testForwarder1)
	srv2.TestModel.AddForwarder(context.Background(), testForwarder2)

	// Register in both
	nseReg := srv2.registerFakeEndpointWithName("golden_network", "test", Worker, "ep1")

	// Add to local endpoints for Server2
	srv2.TestModel.AddEndpoint(context.Background(), nseReg)

	l1 := newTestConnectionModelListener()
	l2 := newTestConnectionModelListener()

	srv.TestModel.AddListener(l1)
	srv2.TestModel.AddListener(l2)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "golden_network",
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					DstIpRequired: true,
					SrcIpRequired: true,
				},
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*networkservice.Mechanism{
			{
				Type: kernel.MECHANISM,
				Parameters: map[string]string{
					common.NetNSInodeKey:    "10",
					common.InterfaceNameKey: "icmp-responder1",
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
	g.Expect(clientConnection1.Xcon.Destination.Mechanism.GetParameters()[vxlan.SrcIP]).To(Equal("127.0.0.1"))

	clientConnection2 := srv2.TestModel.GetClientConnection(clientConnection1.Xcon.Destination.GetId())
	g.Expect(clientConnection2.GetID()).To(Equal("1"))

	timeout := time.Second * 10

	l1.WaitAdd(1, timeout, t)
	// We need to inform cross connection monitor about this connection, since forwarder is fake one.
	epName := clientConnection1.Endpoint.GetNetworkServiceEndpoint().GetName()
	_, err = srv.nseRegistry.RemoveNSE(context.Background(), &registry.RemoveNSERequest{
		NetworkServiceEndpointName: epName,
	})
	if err != nil {
		t.Fatal("Err must be nil")
	}

	// Simulate forwarder dead
	srv.TestModel.AddForwarder(context.Background(), testForwarder1_1)
	srv.TestModel.DeleteForwarder(context.Background(), testForwarder1.RegisteredName)

	// We need to inform cross connection monitor about this connection, since forwarder is fake one.
	// First update is with down state
	// But we want to wait for Up state
	l1.WaitUpdate(5, timeout, t)
	// We need to inform cross connection monitor about this connection, since forwarder is fake one.

	clientConnection1_1 := srv.TestModel.GetClientConnection(nsmResponse.GetId())
	g.Expect(clientConnection1_1 != nil).To(Equal(true))
	g.Expect(clientConnection1_1.GetID()).To(Equal("1"))
	g.Expect(clientConnection1_1.Xcon.Destination.GetId()).To(Equal("1"))
	g.Expect(clientConnection1_1.Xcon.Destination.GetNetworkServiceEndpointName()).To(Equal(epName))
	g.Expect(clientConnection1_1.Xcon.Destination.GetMechanism().GetParameters()[vxlan.SrcIP]).To(Equal("127.0.0.7"))
}
