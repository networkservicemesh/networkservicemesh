package remote

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
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
		Server:  monitor.NewServer(createEvent),
		manager: manager,
	}
	go rv.Serve()
	return rv
}

// MonitorConnections adds recipient for MonitorServer events
func (m *monitorServer) MonitorConnections(selector *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	filtered := newMonitorConnectionFilter(selector, recipient)
	result := m.MonitorEntities(filtered)
	m.manager.UpdateRemoteMonitorDone(selector.NetworkServiceManagerName)
	return result
}
