package tests

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/controlplane/pkg/monitor_crossconnect_server"
	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/pkg/tools"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

func startAPIServer(model model.Model, nsmdApiAddress string) (error, *grpc.Server, monitor_crossconnect_server.MonitorCrossConnectServer) {
	sock, err := net.Listen("tcp", nsmdApiAddress)
	if err != nil {
		return err, nil, nil
	}
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)

	// Start Cross connect monitor and server
	monitor := monitor_crossconnect_server.NewMonitorCrossConnectServer()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitor)
	monitorClient := nsmd.NewMonitorCrossConnectClient(monitor)
	monitorClient.Register(model)
	// TODO: Add more public API services here.

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Errorf("failed to start gRPC NSMD API server %+v", err)
		}
	}()
	logrus.Infof("NSM gRPC API Server: %s is operational", nsmdApiAddress)

	return nil, grpcServer, monitor
}

func TestCCServerEmpty(t *testing.T) {
	RegisterTestingT(t)

	myModel := model.NewModel()

	crossConnectAddress := "127.0.0.1:5007"

	err, grpcServer, monitor := startAPIServer(myModel, crossConnectAddress)
	defer grpcServer.Stop()

	Expect(err).To(BeNil())

	monitor.UpdateCrossConnect(&crossconnect.CrossConnect{
		Id:      "cc1",
		Payload: "json_data",
	})
	events := readNMSDCrossConnectEvents(crossConnectAddress, 1)

	Expect(len(events)).To(Equal(1))

	Expect(events[0].CrossConnects["cc1"].Payload).To(Equal("json_data"))
}

func TestCCServer(t *testing.T) {
	RegisterTestingT(t)

	myModel := model.NewModel()
	crossConnectAddress := "127.0.0.1:5007"

	err, grpcServer, _ := startAPIServer(myModel, crossConnectAddress)
	Expect(err).To(BeNil())
	defer grpcServer.Stop()

	// Now we have CrossConnectMonitor ruunning on default location.
	// We need to have a similated Dataplan via socket for client to connect to.
	dataplaneSocket := "/var/lib/networkservicemesh/nsm.controlplane.dataplane.test.io.sock"
	ln, srv2, monitor2 := createCrossMonitorDataplaneMock(dataplaneSocket)
	// Now we could test it out by adding Dataplane item directory.

	defer ln.Close()
	defer srv2.Stop()

	dataplane := &model.Dataplane{
		RegisteredName: "test_dp",
		SocketLocation: dataplaneSocket,
	}

	myModel.AddDataplane(dataplane)
	// Now it should connect to NSMD CrossConnectMonitor

	// It should pass data via Client to NSMD CrossConnectMonitor, so we could recieve data from it.

	monitor2.UpdateCrossConnect(&crossconnect.CrossConnect{
		Id:      "cc1",
		Payload: "json_data",
	})

	count := 50
	for {
		events := readNMSDCrossConnectEvents(crossConnectAddress, 1)
		if len(events) == 1 {
			if len(events[0].CrossConnects) == 0 {
				time.Sleep(100)
				count--
				if count == 0 {
					return
				}
				continue
			}
		}

		Expect(len(events)).To(Equal(1))
		Expect(events[0].CrossConnects["cc1"].Payload).To(Equal("json_data"))
		break
	}
}

func readNMSDCrossConnectEvents(address string, count int) []*crossconnect.CrossConnectEvent {
	var err error
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", address, err)
		return nil
	}
	defer conn.Close()
	dataplaneClient := crossconnect.NewMonitorCrossConnectClient(conn)

	// Looping indefinetly or until grpc returns an error indicating the other end closed connection.
	stream, err := dataplaneClient.MonitorCrossConnects(context.Background(), &empty.Empty{})
	if err != nil {
		logrus.Warningf("Error: %+v.", err)
		return nil
	}
	pos := 0
	result := []*crossconnect.CrossConnectEvent{}
	for {
		event, err := stream.Recv()
		if err != nil {
			logrus.Errorf("Error2: %+v.", err)
			return result
		}
		result = append(result, event)
		pos++
		if pos == count {
			return result
		}
	}
}

func createCrossMonitorDataplaneMock(dataplaneSocket string) (net.Listener, *grpc.Server, monitor_crossconnect_server.MonitorCrossConnectServer) {
	tools.SocketCleanup(dataplaneSocket)
	ln, err := net.Listen("unix", dataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error listening on socket %s: %s ", dataplaneSocket, err)
	}
	server := grpc.NewServer()
	monitor := monitor_crossconnect_server.NewMonitorCrossConnectServer()
	crossconnect.RegisterMonitorCrossConnectServer(server, monitor)

	go server.Serve(ln)
	return ln, server, monitor
}
