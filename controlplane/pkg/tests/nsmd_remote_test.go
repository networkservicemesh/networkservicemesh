package tests

import (
	"context"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/nsmdapi"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/registry"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"testing"
)

// Below only tests

func TestNSMDRequestClientRemoteNSMD(t *testing.T) {
	RegisterTestingT(t)

	srv, err := newNSMDFullServer()
	Expect(err).To(BeNil())

	srv2, err := newNSMDFullServer()
	Expect(err).To(BeNil())

	srv.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
	})

	srv2.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
	})

	// Register in both
	nseReg := &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    "golden_network",
			Payload: "test",
		},
		NetworkServiceManager: &registry.NetworkServiceManager{
			Name: "my_man",
			Url:  srv2.serviceRegistry.GetPublicAPI(),
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceManagerName: "my_man",
			Payload:                   "test",
			NetworkServiceName:        "golden_network",
			EndpointName:              "golden_network_provider",
		},
	}
	regResp, err := srv.nseRegistry.RegisterNSE(context.Background(), nseReg)
	Expect(err).To(BeNil())
	Expect(regResp.NetworkService.Name).To(Equal("golden_network"))

	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

	// Now we could connect and check RequestClient connection code is working fine.
	// It will use our mock registration service to register itself to.
	// Since we passed our api registry, we could connect using our test service Registry.

	client, con, err := srv.serviceRegistry.NSMDApiClient()
	Expect(err).To(BeNil())
	defer con.Close()

	response, err := client.RequestClientConnection(context.Background(), &nsmdapi.ClientConnectionRequest{})

	Expect(err).To(BeNil())

	logrus.Printf("workspace %s", response.Workspace)

	Expect(response.Workspace).To(Equal("nsm-1"))
	Expect(response.HostBasedir).To(Equal("/var/lib/networkservicemesh/"))

	// Now we could try to connect via Client API
	nsmClient, conn, err := newNetworkServiceClient(response.HostBasedir + "/" + response.Workspace + "/" + response.NsmServerSocket)
	Expect(err).To(BeNil())
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: "golden_network",
			Context: map[string]string{
				"requires": "src_ip,dst_ip",
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
