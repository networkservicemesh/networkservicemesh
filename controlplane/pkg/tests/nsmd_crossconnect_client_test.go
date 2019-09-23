package tests

import (
	"context"
	"net"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/sdk/monitor/crossconnect"
)

type monitorManager struct {
	crossConnectMonitor     monitor_crossconnect.MonitorServer
	remoteConnectionMonitor remote.MonitorServer
	localConnectionMonitors map[string]monitor.Server
}

func (m *monitorManager) CrossConnectMonitor() monitor_crossconnect.MonitorServer {
	return m.crossConnectMonitor
}

func (m *monitorManager) RemoteConnectionMonitor() monitor.Server {
	return m.remoteConnectionMonitor
}

func (m *monitorManager) LocalConnectionMonitor(workspace string) monitor.Server {
	return m.localConnectionMonitors[workspace]
}

type endpointManager struct {
	model model.Model
}

func (stub *endpointManager) DeleteEndpointWithBrokenConnection(endpoint *model.Endpoint) error {
	stub.model.DeleteEndpoint(endpoint.EndpointName())
	return nil
}

func startAPIServer(model model.Model, nsmdApiAddress string) (*grpc.Server, monitor_crossconnect.MonitorServer, net.Listener, error) {
	sock, err := net.Listen("tcp", nsmdApiAddress)
	if err != nil {
		return nil, nil, sock, err
	}
	grpcServer := tools.NewServer()
	serviceRegistry := nsmd.NewServiceRegistry()

	xconManager := services.NewClientConnectionManager(model, nil, serviceRegistry)

	monitorManager := &monitorManager{
		crossConnectMonitor:     monitor_crossconnect.NewMonitorServer(),
		remoteConnectionMonitor: remote.NewMonitorServer(xconManager),
		localConnectionMonitors: map[string]monitor.Server{},
	}

	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitorManager.crossConnectMonitor)
	connection.RegisterMonitorConnectionServer(grpcServer, monitorManager.remoteConnectionMonitor)

	monitorClient := nsmd.NewMonitorCrossConnectClient(model, monitorManager, xconManager, &endpointManager{model: model})
	model.AddListener(monitorClient)
	// TODO: Add more public API services here.

	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Errorf("failed to start gRPC NSMD API server %+v", err)
		}
	}()
	logrus.Infof("NSM gRPC API Server: %s is operational", nsmdApiAddress)

	return grpcServer, monitorManager.crossConnectMonitor, sock, nil
}

func TestCCServerEmpty(t *testing.T) {
	g := NewWithT(t)

	myModel := model.NewModel()

	crossConnectAddress := "127.0.0.1:0"

	grpcServer, monitor, sock, err := startAPIServer(myModel, crossConnectAddress)
	defer grpcServer.Stop()

	crossConnectAddress = sock.Addr().String()

	g.Expect(err).To(BeNil())

	monitor.Update(&crossconnect.CrossConnect{
		Id:      "cc1",
		Payload: "json_data",
	})
	events := readNMSDCrossConnectEvents(crossConnectAddress, 1)

	g.Expect(len(events)).To(Equal(1))

	g.Expect(events[0].CrossConnects["cc1"].Payload).To(Equal("json_data"))
}

func readNMSDCrossConnectEvents(address string, count int) []*crossconnect.CrossConnectEvent {
	var err error
	conn, err := tools.DialTCP(address)
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
