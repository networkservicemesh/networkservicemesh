package tests

import (
	"context"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	connection2 "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
)

// Below only tests

func TestNSMDRequestClientRemoteNSMD(t *testing.T) {
	RegisterTestingT(t)

	srv := newNSMDFullServer()
	srv2 := newNSMDFullServer()
	defer srv.Stop()
	defer srv2.Stop()
	srv.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		RemoteMechanisms: []*connection2.Mechanism{
			&connection2.Mechanism{
				Type: connection2.MechanismType_VXLAN,
				Parameters: map[string]string{
					connection2.VXLANVNI:   "1",
					connection2.VXLANSrcIP: "127.0.0.1",
				},
			},
		},
	})

	srv2.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		RemoteMechanisms: []*connection2.Mechanism{
			&connection2.Mechanism{
				Type: connection2.MechanismType_VXLAN,
				Parameters: map[string]string{
					connection2.VXLANVNI:   "3",
					connection2.VXLANSrcIP: "127.0.0.2",
				},
			},
		},
	})

	// Register in both
	nseReg := srv.registerFakeEndpoint("golden_network", "test", srv2.serviceRegistry.GetPublicAPI())
	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
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
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for cross connections.
	cross_connections := srv2.serviceRegistry.testDataplaneConnection.connections
	Expect(len(cross_connections)).To(Equal(1))
	logrus.Print("End of test")
}

func TestNSMDCloseCrossConnection(t *testing.T) {
	RegisterTestingT(t)

	srv := newNSMDFullServer()
	srv2 := newNSMDFullServer()
	defer srv.Stop()
	defer srv2.Stop()
	srv.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		RemoteMechanisms: []*connection2.Mechanism{
			&connection2.Mechanism{
				Type: connection2.MechanismType_VXLAN,
				Parameters: map[string]string{
					connection2.VXLANVNI:   "1",
					connection2.VXLANSrcIP: "127.0.0.1",
				},
			},
		},
	})

	srv2.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		RemoteMechanisms: []*connection2.Mechanism{
			&connection2.Mechanism{
				Type: connection2.MechanismType_VXLAN,
				Parameters: map[string]string{
					connection2.VXLANVNI:   "3",
					connection2.VXLANSrcIP: "127.0.0.2",
				},
			},
		},
	})

	// Register in both
	nseReg := srv.registerFakeEndpoint("golden_network", "test", srv2.serviceRegistry.GetPublicAPI())
	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
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
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for cross connections.
	cross_connection := srv.testModel.GetClientConnection(nsmResponse.Id)
	Expect(cross_connection).ToNot(BeNil())

	cross_connection2 := srv2.testModel.GetClientConnection(nsmResponse.Id)
	Expect(cross_connection2).ToNot(BeNil())

	//Cross connection successfully created, check it closing
	_, err = nsmClient.Close(context.Background(), nsmResponse)
	Expect(err).To(BeNil())

	//We need to check that xcons have been removed from model
	cross_connection = srv.testModel.GetClientConnection(nsmResponse.Id)
	Expect(cross_connection).To(BeNil())

	cross_connection2 = srv2.testModel.GetClientConnection(nsmResponse.Id)
	Expect(cross_connection2).To(BeNil())

}
