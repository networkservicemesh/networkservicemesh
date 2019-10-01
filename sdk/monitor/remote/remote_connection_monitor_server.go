package remote

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
)

// MonitorServer is a monitor.Server for remote/connection GRPC API
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
func (s *monitorServer) MonitorConnections(selector *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	filtered := newMonitorConnectionFilter(selector, recipient)
	s.MonitorEntities(filtered)
	return nil
}
