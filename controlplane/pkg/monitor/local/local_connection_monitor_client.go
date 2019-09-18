package local

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

type eventStream struct {
	connection.MonitorConnection_MonitorConnectionsClient
}

func (s *eventStream) Recv() (interface{}, error) {
	return s.MonitorConnection_MonitorConnectionsClient.Recv()
}

func newEventStream(ctx context.Context, cc *grpc.ClientConn) (monitor.EventStream, error) {
	stream, err := connection.NewMonitorConnectionClient(cc).MonitorConnections(ctx, &empty.Empty{})

	return &eventStream{
		MonitorConnection_MonitorConnectionsClient: stream,
	}, err
}

// NewMonitorClient creates a new monitor.Client for local/connection GRPC API
func NewMonitorClient(cc *grpc.ClientConn) (monitor.Client, error) {
	return monitor.NewClient(cc, &eventFactory{}, newEventStream)
}
