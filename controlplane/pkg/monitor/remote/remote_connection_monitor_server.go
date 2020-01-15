package remote

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/connectionmonitor"
)

// MonitorServer is a monitor.Server for remote/connection GRPC API
type MonitorServer interface {
	monitor.Server
	connection.MonitorConnectionServer
}

type monitorServer struct {
	connectionmonitor.MonitorServer
	manager *services.ClientConnectionManager
}

// NewMonitorServer creates a new MonitorServer
func NewMonitorServer(manager *services.ClientConnectionManager) MonitorServer {
	rv := &monitorServer{
		MonitorServer: connectionmonitor.NewMonitorServer("RemoteConnection"),
		manager:       manager,
	}
	return rv
}

// MonitorConnections adds recipient for MonitorServer events
func (s *monitorServer) MonitorConnections(selector *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	err := s.MonitorServer.MonitorConnections(selector, recipient)
	if s.manager != nil {
		s.manager.UpdateRemoteMonitorDone(selector)
	}
	return err
}
