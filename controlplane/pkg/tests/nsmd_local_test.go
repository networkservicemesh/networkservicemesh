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

func TestNSMDRequestClientConnectionRequest(t *testing.T) {
	RegisterTestingT(t)

	srv, err := newNSMDFullServer()
	Expect(err).To(BeNil())

	srv.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
	})

	regResp, err := srv.nseRegistry.RegisterNSE(context.Background(), &registry.NSERegistration{
		NetworkService: &registry.NetworkService{
			Name:    "golden_network",
			Payload: "test",
		},
		NetworkServiceManager: &registry.NetworkServiceManager{
			Name: srv.serviceRegistry.GetPublicAPI(),
			Url:  srv.serviceRegistry.GetPublicAPI(),
		},
		NetworkserviceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceManagerName: srv.serviceRegistry.GetPublicAPI(),
			Payload:                   "test",
			NetworkServiceName:        "golden_network",
			EndpointName:              "golden_network_provider",
		},
	})
	Expect(err).To(BeNil())
	Expect(regResp.NetworkService.Name).To(Equal("golden_network"))

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
	logrus.Print("End of test")
}
