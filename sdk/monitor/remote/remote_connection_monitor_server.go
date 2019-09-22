package remote

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
)

// MonitorServer is a monitor.Server for remote/connection GRPC API
type MonitorServer interface {
	monitor.Server
	connection.MonitorConnectionServer
}

type monitorServer struct {
	monitor.Server
	manager *services.ClientConnectionManager
}

// NewMonitorServer creates a new MonitorServer
func NewMonitorServer(manager *services.ClientConnectionManager) MonitorServer {
	rv := &monitorServer{
		Server:  monitor.NewServer(&eventFactory{}),
		manager: manager,
	}
	go rv.Serve()
	return rv
}

// MonitorConnections adds recipient for MonitorServer events
func (s *monitorServer) MonitorConnections(selector *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	filtered := newMonitorConnectionFilter(selector, recipient)
	s.MonitorEntities(filtered)

	s.manager.UpdateRemoteMonitorDone(selector.NetworkServiceManagerName)

	return nil
}
