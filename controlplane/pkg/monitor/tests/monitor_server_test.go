package tests

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect_monitor"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"sync"
	"testing"
)

func startClient(target string) {
	conn, err := grpc.Dial(target, grpc.WithInsecure())
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

	listener, err := net.Listen("tcp", "localhost:5002")
	defer listener.Close()
	Expect(err).To(BeNil())

	grpcServer := grpc.NewServer()
	monitor := crossconnect_monitor.NewCrossConnectMonitor()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitor)

	go func() {
		err := grpcServer.Serve(listener)
		Expect(err).To(BeNil())
	}()

	monitor.Update(&crossconnect.CrossConnect{Id: "1"})

	startClient("localhost:5002")
}

func TestSeveralRecipient(t *testing.T) {
	RegisterTestingT(t)

	listener, err := net.Listen("tcp", "localhost:5002")
	defer listener.Close()
	Expect(err).To(BeNil())

	grpcServer := grpc.NewServer()
	monitor := crossconnect_monitor.NewCrossConnectMonitor()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitor)

	go func() {
		err := grpcServer.Serve(listener)
		Expect(err).To(BeNil())
	}()

	monitor.Update(&crossconnect.CrossConnect{Id: "1"})

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			startClient("localhost:5002")
			wg.Done()
		}()
	}
	wg.Wait()
}
