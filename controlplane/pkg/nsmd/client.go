package nsmd

import (
	"context"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/local/networkservice"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"google.golang.org/grpc"
	"os"
	"time"
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

	return tools.SocketOperationCheck(nsmServerSocket)
}
