package monitor_crossconnect_server

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/controlplane/pkg/model"
	"github.com/ligato/networkservicemesh/pkg/tools"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

func TestCCServerEmpty(t *testing.T) {
	RegisterTestingT(t)

	myModel := model.NewModel("127.0.0.1:5000")

	crossConnectAddress := "127.0.0.1:5007"

	err, grpcServer, monitor := StartNSMCrossConnectServer(myModel, crossConnectAddress)
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

	myModel := model.NewModel("127.0.0.1:5000")
	crossConnectAddress := "127.0.0.1:5007"

	err, grpcServer, _ := StartNSMCrossConnectServer(myModel, crossConnectAddress)
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
	conn, err := dial(context.Background(), "tcp", address)
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

func createCrossMonitorDataplaneMock(dataplaneSocket string) (net.Listener, *grpc.Server, MonitorCrossConnectServer) {
	tools.SocketCleanup(dataplaneSocket)
	ln, err := net.Listen("unix", dataplaneSocket)
	if err != nil {
		logrus.Fatalf("Error listening on socket %s: %s ", dataplaneSocket, err)
	}
	server := grpc.NewServer()
	monitor := NewMonitorCrossConnectServer()
	crossconnect.RegisterMonitorCrossConnectServer(server, monitor)

	go server.Serve(ln)
	return ln, server, monitor
}
