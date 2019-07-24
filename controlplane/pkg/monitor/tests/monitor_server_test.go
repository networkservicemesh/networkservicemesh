package tests

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
)

func startClient(target string) {
	conn, err := tools.DialTCP(target)
	defer conn.Close()

	Expect(err).To(BeNil())
	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	stream, err := monitorClient.MonitorCrossConnects(context.Background(), &empty.Empty{})
	Expect(err).To(BeNil())

	event, err := stream.Recv()
	Expect(err).To(BeNil())
	logrus.Infof("Receive event: %v", event)
	Expect(event).NotTo(BeNil())
	Expect(event.Type).To(Equal(crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER))
	Expect(event.CrossConnects).NotTo(BeNil())
	Expect(event.CrossConnects["1"]).NotTo(BeNil())
}

func TestSimple(t *testing.T) {
	RegisterTestingT(t)

	listener, err := net.Listen("tcp", "localhost:0")
	defer listener.Close()
	Expect(err).To(BeNil())

	grpcServer := grpc.NewServer()
	monitor := monitor_crossconnect.NewMonitorServer()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitor)

	go func() {
		grpcServer.Serve(listener)
	}()

	monitor.Update(&crossconnect.CrossConnect{Id: "1"})

	startClient(listenerAddress(listener))
}

func listenerAddress(listener net.Listener) string {
	port := listener.Addr().(*net.TCPAddr).Port
	logrus.Infof("Connect to port: %d", port)
	return fmt.Sprintf("localhost:%d", port)
}

func TestSeveralRecipient(t *testing.T) {
	RegisterTestingT(t)

	listener, err := net.Listen("tcp", "localhost:0")
	defer listener.Close()
	Expect(err).To(BeNil())

	grpcServer := grpc.NewServer()
	monitor := monitor_crossconnect.NewMonitorServer()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitor)

	var waitServe sync.WaitGroup
	waitServe.Add(1)
	go func() {
		go waitServe.Done()
		_ = grpcServer.Serve(listener)
	}()
	waitServe.Wait()
	monitor.Update(&crossconnect.CrossConnect{Id: "1"})

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			startClient(listenerAddress(listener))
			wg.Done()
		}()
	}
	wg.Wait()
	logrus.Infof("######END")
}
