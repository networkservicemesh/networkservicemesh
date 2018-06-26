package main

import (
	"context"
	"flag"
	"net"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"google.golang.org/grpc"
)

const (
	// clientConnectionTimeout defines time the client waits for establishing connection with the server
	clientConnectionTimeout = time.Second * 60
)

var (
	clientSocket = flag.String("nsm-socket", "/var/lib/networkservicemesh/nsm.ligato.io.sock", "Location of NSM process client access socket")
)

func dial(ctx context.Context, unixSocketPath string) (*grpc.ClientConn, error) {
	c, err := grpc.DialContext(ctx, unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	return c, err
}

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")

	// Checking if NSM Client socket exists and of not crash init container
	_, err := os.Stat(*clientSocket)
	if err != nil {
		glog.Errorf("NSM Client: Failure to access NSM socket at %s with error: %+v, existing...", *clientSocket, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), clientConnectionTimeout)
	conn, err := dial(ctx, *clientSocket)
	if err != nil {
		glog.Errorf("NSM Client: Failure to communicate with the socket %s with error: %+v", *clientSocket, err)
		os.Exit(1)
	}
	nsmClient := nsmconnect.NewClientConnectionClient(conn)
	defer conn.Close()
	defer cancel()
	glog.Infof("NSM Client: Connection to NSM server on socket: %s succeeded.", *clientSocket)
	glog.Infof("NSM Client: Client API %+v", nsmClient)
	// Init related activities start here

	// Init related activities start here
	glog.Info("NSM Client: Initialization is completed successfully, exiting...")
	os.Exit(0)
}
