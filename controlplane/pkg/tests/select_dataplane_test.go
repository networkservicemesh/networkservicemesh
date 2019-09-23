package tests

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsm/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

func createTestDataplane(name string, localMechanisms, remoteMechanisms []connection.Mechanism) *model.Dataplane {
	return &model.Dataplane{
		RegisteredName:       name,
		SocketLocation:       "tcp:some_addr",
		LocalMechanisms:      localMechanisms,
		RemoteMechanisms:     remoteMechanisms,
		MechanismsConfigured: true,
	}
}

func TestSelectDataplane(t *testing.T) {
	g := NewWithT(t)

	testDataplane1_1 := createTestDataplane("test_data_plane_2",
		[]connection.Mechanism{
			&local.Mechanism{
				Type: local.MechanismType_VHOST_INTERFACE,
			},
			&local.Mechanism{
				Type: local.MechanismType_MEM_INTERFACE,
			},
		},
		[]connection.Mechanism{
			&remote.Mechanism{
				Type: remote.MechanismType_VXLAN,
				Parameters: map[string]string{
					remote.VXLANSrcIP: "127.0.0.1",
				},
			},
			&remote.Mechanism{
				Type: remote.MechanismType_GRE,
				Parameters: map[string]string{
					remote.VXLANSrcIP: "127.0.0.1",
				},
			},
		})

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	srv2 := NewNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()
	srv.TestModel.AddDataplane(testDataplane1)
	srv2.TestModel.AddDataplane(testDataplane2)
	srv.TestModel.AddDataplane(testDataplane1_1)

	// Register in both
	nseReg := srv2.RegisterFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.TestModel.AddEndpoint(nseReg)

	l1 := newTestConnectionModelListener(Master)
	l2 := newTestConnectionModelListener(Worker)

	srv.TestModel.AddListener(l1)
	srv2.TestModel.AddListener(l2)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &local.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					DstIpRequired: true,
					SrcIpRequired: true,
				},
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*local.Mechanism{
			{
				Type: local.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					local.NetNsInodeKey:    "10",
					local.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	request = &networkservice.NetworkServiceRequest{
		Connection: &local.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					DstIpRequired: true,
					SrcIpRequired: true,
				},
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*local.Mechanism{
			{
				Type: local.MechanismType_MEM_INTERFACE,
				Parameters: map[string]string{
					local.NetNsInodeKey:    "10",
					local.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err = nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	request = &networkservice.NetworkServiceRequest{
		Connection: &local.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					DstIpRequired: true,
					SrcIpRequired: true,
				},
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*local.Mechanism{
			{
				Type: local.MechanismType_SRIOV_INTERFACE,
				Parameters: map[string]string{
					local.NetNsInodeKey:    "10",
					local.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err = nsmClient.Request(context.Background(), request)
	g.Expect(err).NotTo(BeNil())
	g.Expect(err.Error()).To(ContainSubstring("no appropriate dataplanes found"))
}
