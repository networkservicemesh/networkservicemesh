package local

import (
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

// MonitorServer is a monitor.Server for local/connection GRPC API
type MonitorServer interface {
	monitor.Server
	connection.MonitorConnectionServer
}

type monitorServer struct {
	monitor.Server
}

// NewMonitorServer creates a new MonitorServer
func NewMonitorServer() MonitorServer {
	rv := &monitorServer{
		Server: monitor.NewServer(&eventFactory{}),
	}
	go rv.Serve()
	return rv
}

// MonitorConnections adds recipient for MonitorServer events
func (s *monitorServer) MonitorConnections(_ *empty.Empty, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	s.MonitorEntities(recipient)
	return nil
}
