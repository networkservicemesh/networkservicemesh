package remote_connection_monitor

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/services"
)

type RemoteConnectionMonitor struct {
	monitor.MonitorServer
	manager *services.ClientConnectionManager
}

func NewRemoteConnectionMonitor(manager *services.ClientConnectionManager) *RemoteConnectionMonitor {
	rv := &RemoteConnectionMonitor{
		MonitorServer: monitor.NewMonitorServer(CreateRemoteConnectionEvent),
		manager:       manager,
	}
	go rv.Serve()
	return rv
}

func (m *RemoteConnectionMonitor) MonitorConnections(selector *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	filtered := NewMonitorConnectionFilter(selector, recipient)
	result := m.MonitorEntities(filtered)
	m.manager.UpdateRemoteMonitorDone(selector.NetworkServiceManagerName)
	return result
}
