package tests

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	localConnection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	localNetworkservice "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	remoteConnection "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	. "github.com/onsi/gomega"
)

import (
	"context"
	"testing"
)

func createTestDataplane(name string, localMechanisms []*localConnection.Mechanism, RemoteMechanisms []*remoteConnection.Mechanism) *model.Dataplane {
	return &model.Dataplane{
		RegisteredName:   name,
		SocketLocation:   "tcp:some_addr",
		LocalMechanisms:  localMechanisms,
		RemoteMechanisms: RemoteMechanisms,
		MechanismsConfigured: true,
	}
}

func TestSelectDataplane(t *testing.T) {
	RegisterTestingT(t)

	testDataplane1_1 := createTestDataplane("test_data_plane_2",
		[]*localConnection.Mechanism{
			{
				Type: localConnection.MechanismType_VHOST_INTERFACE,
			},
			{
				Type: localConnection.MechanismType_MEM_INTERFACE,
			},
		},
		[]*remoteConnection.Mechanism{
			{
				Type: remoteConnection.MechanismType_VXLAN,
				Parameters: map[string]string{
					remoteConnection.VXLANSrcIP: "127.0.0.1",
				},
			},
			{
				Type: remoteConnection.MechanismType_GRE,
				Parameters: map[string]string{
					remoteConnection.VXLANSrcIP: "127.0.0.1",
				},
			},
		})

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage, defaultClusterConfiguration)
	srv2 := newNSMDFullServer(Worker, storage, defaultClusterConfiguration)
	defer srv.Stop()
	defer srv2.Stop()
	srv.testModel.AddDataplane(testDataplane1)
	srv2.testModel.AddDataplane(testDataplane2)
	srv.testModel.AddDataplane(testDataplane1_1)

	// Register in both
	nseReg := srv2.registerFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

	l1 := newTestConnectionModelListener()
	l2 := newTestConnectionModelListener()

	srv.testModel.AddListener(l1)
	srv2.testModel.AddListener(l2)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &localNetworkservice.NetworkServiceRequest{
		Connection: &localConnection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*localConnection.Mechanism{
			{
				Type: localConnection.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					localConnection.NetNsInodeKey:    "10",
					localConnection.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	request = &localNetworkservice.NetworkServiceRequest{
		Connection: &localConnection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*localConnection.Mechanism{
			{
				Type: localConnection.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					localConnection.NetNsInodeKey:    "10",
					localConnection.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err = nsmClient.Request(context.Background(), request)
	Expect(err).To(BeNil())
	Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	request = &localNetworkservice.NetworkServiceRequest{
		Connection: &localConnection.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				DstIpRequired: true,
				SrcIpRequired: true,
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*localConnection.Mechanism{
			{
				Type: localConnection.MechanismType_SRIOV_INTERFACE,
				Parameters: map[string]string{
					localConnection.NetNsInodeKey:    "10",
					localConnection.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err = nsmClient.Request(context.Background(), request)
	Expect(err).NotTo(BeNil())
	Expect(err.Error()).To(ContainSubstring("no appropriate dataplanes found"))
}
