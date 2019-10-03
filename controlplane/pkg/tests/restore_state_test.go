package tests

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
)

func TestRestoreConnectionState(t *testing.T) {
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	defer srv.Stop()

	srv.AddFakeDataplane("dp1", "tcp:some_address")

	g.Expect(srv.nsmServer.Manager().WaitForDataplane(context.Background(), 1*time.Millisecond).Error()).To(Equal("Failed to wait for NSMD stare restore... timeout 1ms happened"))

	xcons := []*crossconnect.CrossConnect{}
	srv.nsmServer.Manager().RestoreConnections(xcons, "dp1")
	g.Expect(srv.nsmServer.Manager().WaitForDataplane(context.Background(), 1*time.Second)).To(BeNil())
}

func TestRestoreConnectionStateWrongDst(t *testing.T) {
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	defer srv.Stop()

	srv.AddFakeDataplane("dp1", "tcp:some_address")
	srv.registerFakeEndpointWithName("ns1", "IP", Worker, "ep2")

	nsmClient := srv.RequestNSM("nsm1")

	requestConnection := &connection.Connection{
		Id:             "1",
		NetworkService: "ns1",
		Mechanism: &connection.Mechanism{
			Parameters: map[string]string{
				connection.Workspace: nsmClient.Workspace,
			},
		},
	}

	dstConnection := &connection.Connection{
		Id: "2",
		Mechanism: &connection.Mechanism{
			Type: connection.MechanismType_KERNEL_INTERFACE,
			Parameters: map[string]string{
				connection.WorkspaceNSEName: "nse1",
			},
		},
		NetworkService: "ns1",
	}
	xcons := []*crossconnect.CrossConnect{
		&crossconnect.CrossConnect{
			Source: &crossconnect.CrossConnect_LocalSource{
				LocalSource: requestConnection,
			},
			Destination: &crossconnect.CrossConnect_LocalDestination{
				LocalDestination: dstConnection,
			},
			Id: "1",
		},
	}
	srv.nsmServer.Manager().RestoreConnections(xcons, "dp1")
	g.Expect(srv.nsmServer.Manager().WaitForDataplane(context.Background(), 1*time.Second)).To(BeNil())
	g.Expect(len(srv.TestModel.GetAllClientConnections())).To(Equal(0))
}
