package tests

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/sdk/monitor/crossconnect"
)

func startClient(g *WithT, target string, ids ...string) {
	visit := map[string]bool{}
	for _, id := range ids {
		visit[id] = false
	}
	timeout := time.After(time.Second)
	logrus.Infof("Expected cross connects with ids = %v", ids)
	_ = os.Setenv(tools.InsecureEnv, "true")
	conn, err := tools.DialTCP(target)
	defer conn.Close()

	g.Expect(err).To(BeNil())
	monitorClient := crossconnect.NewMonitorCrossConnectClient(conn)
	stream, err := monitorClient.MonitorCrossConnects(context.Background(), &empty.Empty{})
	g.Expect(err).To(BeNil())
	for {
		select {
		case <-timeout:
			logrus.Error("timeout for wait events")
			g.Expect(true).Should(BeFalse())
		default:
			event, err := stream.Recv()
			g.Expect(err).To(BeNil())
			logrus.Infof("Receive event: %v", event)
			g.Expect(event).NotTo(BeNil())
			for _, crossconnect := range event.CrossConnects {
				visit[crossconnect.Id] = true
			}
			exit := true
			for _, v := range visit {
				if !v {
					exit = false
				}
			}
			if exit {
				return
			}
		}
	}
}

func TestEachClientShouldGetAllEvents(t *testing.T) {
	g := NewWithT(t)

	listener, err := net.Listen("tcp", "localhost:0")
	defer listener.Close()
	g.Expect(err).To(BeNil())

	grpcServer := grpc.NewServer()
	monitor := monitor_crossconnect.NewMonitorServer()
	crossconnect.RegisterMonitorCrossConnectServer(grpcServer, monitor)

	go func() {
		grpcServer.Serve(listener)
	}()
	ids := []string{}
	for id := 0; id < 10; id++ {
		monitor.Update(context.Background(), &crossconnect.CrossConnect{Id: fmt.Sprint(id)})
		ids = append(ids, fmt.Sprint(id))
		startClient(g, listenerAddress(listener), ids...)
	}
}

func listenerAddress(listener net.Listener) string {
	port := listener.Addr().(*net.TCPAddr).Port
	logrus.Infof("Connect to port: %d", port)
	return fmt.Sprintf("localhost:%d", port)
}

func TestSeveralRecipient(t *testing.T) {
	g := NewWithT(t)

	listener, err := net.Listen("tcp", "localhost:0")
	defer listener.Close()
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
			startClient(g, listenerAddress(listener), "1")
			wg.Done()
		}()
	}
	wg.Wait()
	logrus.Infof("######END")
}
