package tests

import (
	"context"
	"net"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote_connection_monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

func startAPIServer(model model.Model, nsmdApiAddress string) (error, *grpc.Server, *crossconnect_monitor.CrossConnectMonitor, net.Listener) {
	sock, err := net.Listen("tcp", nsmdApiAddress)
	if err != nil {
		return err, nil, nil, sock
	}
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	serviceRegistry := nsmd.NewServiceRegistry()

	// Start Cross connect monitor and server
	monitor := crossconnect_monitor.NewCrossConnectMonitor()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitor)

	manager := services.NewClientConnectionManager(model, nil, serviceRegistry)
	connectionMonitor := remote_connection_monitor.NewRemoteConnectionMonitor(manager)
	connection.RegisterMonitorConnectionServer(grpcServer, connectionMonitor)

	monitorClient := nsmd.NewMonitorCrossConnectClient(monitor, connectionMonitor, manager)
	model.AddListener(monitorClient)
	// TODO: Add more public API services here.

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Errorf("failed to start gRPC NSMD API server %+v", err)
		}
	}()
	logrus.Infof("NSM gRPC API Server: %s is operational", nsmdApiAddress)

	return nil, grpcServer, monitor, sock
}

func TestCCServerEmpty(t *testing.T) {
	RegisterTestingT(t)

	myModel := model.NewModel()

	crossConnectAddress := "127.0.0.1:0"

	err, grpcServer, monitor, sock := startAPIServer(myModel, crossConnectAddress)
	defer grpcServer.Stop()

	crossConnectAddress = sock.Addr().String()

	Expect(err).To(BeNil())

	monitor.Update(&crossconnect.CrossConnect{
		Id:      "cc1",
		Payload: "json_data",
	})
	events := readNMSDCrossConnectEvents(crossConnectAddress, 1)

	Expect(len(events)).To(Equal(1))

	Expect(events[0].CrossConnects["cc1"].Payload).To(Equal("json_data"))
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

	// Looping indefinitely or until grpc returns an error indicating the other end closed connection.
	stream, err := dataplaneClient.MonitorCrossConnects(context.Background(), &empty.Empty{})
	if err != nil {
		logrus.Warningf("Error: %+v.", err)
		return nil
	}
	pos := 0
	result := []*crossconnect.CrossConnectEvent{}
	for {
		event, err := stream.Recv()
		logrus.Infof("(test) receive event: %s %v", event.Type, event.CrossConnects)
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

func createCrossMonitorDataplaneMock(dataplaneSocket string) (net.Listener, *grpc.Server, *crossconnect_monitor.CrossConnectMonitor) {
	tools.SocketCleanup(dataplaneSocket)
	ln, err := net.Listen("unix", dataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error listening on socket %s: %s ", dataplaneSocket, err)
	}
	server := grpc.NewServer()
	monitor := crossconnect_monitor.NewCrossConnectMonitor()
	crossconnect.RegisterMonitorCrossConnectServer(server, monitor)

	go server.Serve(ln)
	return ln, server, monitor
}
