package remoteconnectionmonitor

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
)

// Server is a monitor.Server for remoteconnection GRPC API
type Server struct {
	monitor.Server
	manager *services.ClientConnectionManager
}

// NewServer creates a new Server
func NewServer(manager *services.ClientConnectionManager) *Server {
	rv := &Server{
		Server:  monitor.NewServer(createEvent),
		manager: manager,
	}
	go rv.Serve()
	return rv
}

// MonitorConnections adds recipient for Server events
func (m *Server) MonitorConnections(selector *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	filtered := newMonitorConnectionFilter(selector, recipient)
	result := m.MonitorEntities(filtered)
	m.manager.UpdateRemoteMonitorDone(selector.NetworkServiceManagerName)
	return result
}
