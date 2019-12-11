// +build unit_test

package tests

import (
	"context"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/memif"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/common"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

func createTestForwarder(name string, localMechanisms, remoteMechanisms []*connection.Mechanism) *model.Forwarder {
	return &model.Forwarder{
		RegisteredName:       name,
		SocketLocation:       "tcp:some_addr",
		LocalMechanisms:      localMechanisms,
		RemoteMechanisms:     remoteMechanisms,
		MechanismsConfigured: true,
	}
}

func TestSelectForwarder(t *testing.T) {
	g := NewWithT(t)

	testForwarder1_1 := createTestForwarder("test_data_plane_2",
		[]*connection.Mechanism{
			{
				Type: "VHOST_INTERFACE",
			},
			{
				Type: memif.MECHANISM,
			},
		},
		[]*connection.Mechanism{
			{
				Type: vxlan.MECHANISM,
				Parameters: map[string]string{
					vxlan.SrcIP: "127.0.0.1",
				},
			},
			{
				Type: "GRE",
				Parameters: map[string]string{
					"dst-ip": "127.0.0.1",
				},
			},
		})

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	srv2 := NewNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()
	srv.TestModel.AddForwarder(context.Background(), testForwarder1)
	srv2.TestModel.AddForwarder(context.Background(), testForwarder2)
	srv.TestModel.AddForwarder(context.Background(), testForwarder1_1)

	// Register in both
	nseReg := srv2.RegisterFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.TestModel.AddEndpoint(context.Background(), nseReg)

	l1 := newTestConnectionModelListener()
	l2 := newTestConnectionModelListener()

	srv.TestModel.AddListener(l1)
	srv2.TestModel.AddListener(l2)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	t.Run("Check-kernel", func(t *testing.T) {
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
					Type: kernel.MECHANISM,
					Parameters: map[string]string{
						common.NetNsInodeKey:    "10",
						common.InterfaceNameKey: "icmp-responder1",
					},
				},
			},
		}
		nsmResponse, err := nsmClient.Request(context.Background(), request)
		g.Expect(err).To(BeNil())
		g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
	})

	t.Run("Check-memif", func(t *testing.T) {
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
					Type: memif.MECHANISM,
					Parameters: map[string]string{
						common.NetNsInodeKey:    "10",
						common.InterfaceNameKey: "icmp-responder1",
					},
				},
			},
		}

		nsmResponse, err := nsmClient.Request(context.Background(), request)
		g.Expect(err).To(BeNil())
		g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))
	})

	t.Run("Check-sriov", func(t *testing.T) {
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
					Type: "SRIOV_INTERFACE",
					Parameters: map[string]string{
						common.NetNsInodeKey:    "10",
						common.InterfaceNameKey: "icmp-responder1",
					},
				},
			},
		}

		nsmResponse, err := nsmClient.Request(context.Background(), request)
		g.Expect(err).NotTo(BeNil())
		g.Expect(nsmResponse).To(BeNil())
		g.Expect(err.Error()).To(ContainSubstring("no appropriate forwarders found"))
	})
}
