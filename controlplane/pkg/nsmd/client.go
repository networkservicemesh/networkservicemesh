package nsmd

import (
	"context"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

func NewNetworkServiceClient() (networkservice.NetworkServiceClient, *grpc.ClientConn, error) {
	conn, err := newNetworkServiceClientSocket()
	if err != nil {
		return nil, nil, err
	}
	// Init related activities start here
	nsmConnectionClient := networkservice.NewNetworkServiceClient(conn)
	return nsmConnectionClient, conn, nil
}

func newNetworkServiceClientSocket() (*grpc.ClientConn, error) {
	nsmServerSocket, _ := os.LookupEnv(NsmServerSocketEnv)
	if _, err := os.Stat(nsmServerSocket); err != nil {
		return nil, err
	}

	// Wait till we actually have an nsmd to talk to
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := tools.WaitForPortAvailable(ctx, "unix", nsmServerSocket, 100*time.Millisecond)
	if err != nil {
		return nil, err
	}

	return tools.DialUnix(nsmServerSocket)
}
