package nsmd

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/remote"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const sendPeriod = time.Millisecond * 15
const testingTime = time.Second / 4

func TestNsmdMonitorShouldHandleServerShutdown(t *testing.T) {
	port, stop := setupServer(t)
	go func() {
		<-time.After(testingTime / 2)
		stop()
	}()
	err := startClient(t, testingTime, port)
	if err == nil {
		t.Fatal("client should detect server shutdown")
	}
	logrus.Info(err)
}

func startClient(t *testing.T, timeout time.Duration, serverPort int) error {
	conn, err := tools.DialTCP(fmt.Sprintf(":%v", serverPort))
	if err != nil {
		t.Fatal(err.Error())
	}
	client, err := connectionmonitor.NewMonitorClient(conn, &connection.MonitorScopeSelector{})
	if err != nil {
		t.Fatal(err.Error())
	}
	for {
		select {
		case <-time.After(timeout):
			return nil
		case result := <-client.ErrorChannel():
			return result
		case event := <-client.EventChannel():
			if event != nil {
				logrus.Info(event.Message())
			}
		}
	}
}

func setupServer(t *testing.T) (int, func()) {
	grcServer := tools.NewServer(context.Background())
	remoteMonitor := remote.NewMonitorServer(nil)
	connection.RegisterMonitorConnectionServer(grcServer, remoteMonitor)
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err.Error())
	}
	go func() {
		grcServer.Serve(l)
	}()
	go func() {
		for {
			<-time.After(sendPeriod)
			remoteMonitor.SendAll(&testEvent{})
		}
	}()

	return l.Addr().(*net.TCPAddr).Port,
		grcServer.Stop
}

type testEvent struct {
}

func (*testEvent) EventType() monitor.EventType {
	return monitor.EventTypeUpdate
}

func (*testEvent) Entities() map[string]monitor.Entity {
	return nil
}

func (t *testEvent) Message() (interface{}, error) {
	return &connection.ConnectionEvent{
		Type:        connection.ConnectionEventType_UPDATE,
		Connections: nil,
	}, nil
}

func (*testEvent) Context() context.Context {
	return context.Background()
}
