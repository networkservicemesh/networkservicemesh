package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func main() {
	// Capture signals to cleanup before exiting
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	var err error
	conn, err := dial(context.Background(), "tcp", "127.0.0.1:5007")
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", "127.0.0.1:5007", err)
		return
	}
	defer conn.Close()
	client := crossconnect.NewMonitorCrossConnectClient(conn)
	// Looping indefinetly or until grpc returns an error indicating the other end closed connection.
	stream, err := client.MonitorCrossConnects(context.Background(), &empty.Empty{})
	if err != nil {
		logrus.Warningf("Error: %+v.", err)
		return
	}
	result := []*crossconnect.CrossConnectEvent{}
	for {
		event, err := stream.Recv()
		if err != nil {
			logrus.Errorf("Error: %+v.", err)
			return
		}
		logrus.Infof("Events: %+v", event)
		result = append(result, event)
	}
}

func dial(ctx context.Context, network string, address string) (*grpc.ClientConn, error) {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.Dial(network, addr)
		}),
	)
	return conn, err
}
