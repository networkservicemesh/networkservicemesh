package tests

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/sdk/monitor/crossconnect"
)

func startClient(g *WithT, target string) {
	_ = os.Setenv(tools.InsecureEnv, "true")
	conn, err := tools.DialTCP(target)
	defer func() {
		if err := conn.Close(); err != nil {
			logrus.Error(err)
		}
	}()

	g.Expect(err).To(BeNil())
	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	stream, err := monitorClient.MonitorCrossConnects(context.Background(), &empty.Empty{})
	g.Expect(err).To(BeNil())

	event, err := stream.Recv()
	g.Expect(err).To(BeNil())
	logrus.Infof("Receive event: %v", event)
	g.Expect(event).NotTo(BeNil())
	g.Expect(event.Type).To(Equal(crossconnect.CrossConnectEventType_INITIAL_STATE_TRANSFER))
	g.Expect(event.CrossConnects).NotTo(BeNil())
	g.Expect(event.CrossConnects["1"]).NotTo(BeNil())
}

func TestSimple(t *testing.T) {
	g := NewWithT(t)

	listener, err := net.Listen("tcp", "localhost:0")
	defer func() {
		if err := listener.Close(); err != nil {
			logrus.Error(err)
		}
	}()
	g.Expect(err).To(BeNil())

	grpcServer := grpc.NewServer()
	monitor := monitor_crossconnect.NewMonitorServer()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitor)

	done := false
	go func() {
		serveErr := grpcServer.Serve(listener)
		if !done {
			assert.NoError(t, serveErr)
		}
	}()

	monitor.Update(context.Background(), &crossconnect.CrossConnect{Id: "1"})

	startClient(g, listenerAddress(listener))
	done = true
}

func listenerAddress(listener net.Listener) string {
	port := listener.Addr().(*net.TCPAddr).Port
	logrus.Infof("Connect to port: %d", port)
	return fmt.Sprintf("localhost:%d", port)
}

func TestSeveralRecipient(t *testing.T) {
	g := NewWithT(t)

	listener, err := net.Listen("tcp", "localhost:0")
	defer func() {
		if listenErr := listener.Close(); listenErr != nil {
			logrus.Error(err)
		}
	}()
	g.Expect(err).To(BeNil())

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
	monitor.Update(context.Background(), &crossconnect.CrossConnect{Id: "1"})

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			startClient(g, listenerAddress(listener))
			wg.Done()
		}()
	}
	wg.Wait()
	logrus.Infof("######END")
}
