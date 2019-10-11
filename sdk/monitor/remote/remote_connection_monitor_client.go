package remote

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/sdk/compat"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
)

type eventStream struct {
	connection.MonitorConnection_MonitorConnectionsClient
}

func (s *eventStream) Recv() (interface{}, error) {
	return s.MonitorConnection_MonitorConnectionsClient.Recv()
}

func newEventStream(ctx context.Context, cc *grpc.ClientConn, selector *connection.MonitorScopeSelector) (monitor.EventStream, error) {
	stream, err := compat.NewRemoteMonitorAdapter(cc).MonitorConnections(ctx, selector)

	return &eventStream{
		MonitorConnection_MonitorConnectionsClient: stream,
	}, err
}

// NewMonitorClient creates a new monitor.Client for remote/connection GRPC API
func NewMonitorClient(cc *grpc.ClientConn, selector *connection.MonitorScopeSelector) (monitor.Client, error) {
	streamConstructor := func(ctx context.Context, grpcCC *grpc.ClientConn) (stream monitor.EventStream, e error) {
		return newEventStream(ctx, grpcCC, selector)
	}

	return monitor.NewClient(cc, &eventFactory{}, streamConstructor)
}
