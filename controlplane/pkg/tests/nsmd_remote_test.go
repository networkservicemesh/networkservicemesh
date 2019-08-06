package tests

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

// Below only tests

func TestNSMDRequestClientRemoteNSMD(t *testing.T) {
	g := NewWithT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	srv2 := newNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()

	srv.testModel.AddDataplane(testDataplane1)

	srv2.testModel.AddDataplane(testDataplane2)

	// Register in both
	nseReg := srv2.registerFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

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

	reply, err := nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(reply.GetConnection().GetNetworkService()).To(Equal("golden_network"))

	// We need to check for cross connections.
	cross_connections := srv2.serviceRegistry.testDataplaneConnection.connections
	g.Expect(len(cross_connections)).To(Equal(1))
	logrus.Print("End of test")
}

func TestNSMDCloseCrossConnection(t *testing.T) {
	g := NewWithT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	srv2 := newNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()
	srv.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		LocalMechanisms: []connection.Mechanism{
			&local.Mechanism{
				Type: local.MechanismType_KERNEL_INTERFACE,
			},
		},
		RemoteMechanisms: []connection.Mechanism{
			&remote.Mechanism{
				Type: remote.MechanismType_VXLAN,
				Parameters: map[string]string{
					remote.VXLANVNI:   "1",
					remote.VXLANSrcIP: "10.1.1.1",
				},
			},
		},
		MechanismsConfigured: true,
	})

	srv2.testModel.AddDataplane(&model.Dataplane{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		RemoteMechanisms: []connection.Mechanism{
			&remote.Mechanism{
				Type: remote.MechanismType_VXLAN,
				Parameters: map[string]string{
					remote.VXLANVNI:   "3",
					remote.VXLANSrcIP: "10.1.1.2",
				},
			},
		},
		MechanismsConfigured: true,
	})

	// Register in both
	nseReg := srv2.registerFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

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

	reply, err := nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(reply.GetConnection().GetNetworkService()).To(Equal("golden_network"))

	// We need to check for cross connections.
	cross_connection := srv.testModel.GetClientConnection(reply.GetConnection().GetId())
	g.Expect(cross_connection).ToNot(BeNil())

	destConnectionId := cross_connection.Xcon.GetRemoteDestination().GetId()

	cross_connection2 := srv2.testModel.GetClientConnection(destConnectionId)
	g.Expect(cross_connection2).ToNot(BeNil())

	//Cross connection successfully created, check it closing
	_, err = nsmClient.Close(context.Background(), reply.GetConnection())
	g.Expect(err).To(BeNil())

	//We need to check that xcons have been removed from model
	cross_connection = srv.testModel.GetClientConnection(reply.GetConnection().GetId())
	g.Expect(cross_connection).To(BeNil())

	cross_connection2 = srv2.testModel.GetClientConnection(destConnectionId)
	g.Expect(cross_connection2).To(BeNil())

}

func TestNSMDDelayRemoteMechanisms(t *testing.T) {
	g := NewWithT(t)

	storage := newSharedStorage()
	srv := newNSMDFullServer(Master, storage)
	srv2 := newNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()

	srv.testModel.AddDataplane(testDataplane1)

	testDataplane2_2 := &model.Dataplane{
		RegisteredName: "test_data_plane2",
		SocketLocation: "tcp:some_addr",
	}

	srv2.testModel.AddDataplane(testDataplane2_2)

	// Register in both
	nseReg := srv2.registerFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.testModel.AddEndpoint(nseReg)

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

	type Response struct {
		nsmResponse *local.Connection
		err         error
	}
	resultChan := make(chan *Response, 1)

	go func(ctx context.Context, req *networkservice.NetworkServiceRequest) {
		reply, err := nsmClient.Request(ctx, req)
		resultChan <- &Response{nsmResponse: reply.GetConnection(), err: err}
	}(context.Background(), request)

	<-time.After(1 * time.Second)

	testDataplane2_2.LocalMechanisms = testDataplane2.LocalMechanisms
	testDataplane2_2.RemoteMechanisms = testDataplane2.RemoteMechanisms
	testDataplane2_2.MechanismsConfigured = true
	srv2.testModel.UpdateDataplane(testDataplane2_2)

	res := <-resultChan
	g.Expect(res.err).To(BeNil())
	g.Expect(res.nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for crМфвук31oss connections.
	cross_connections := srv2.serviceRegistry.testDataplaneConnection.connections
	g.Expect(len(cross_connections)).To(Equal(1))
	logrus.Print("End of test")
}
