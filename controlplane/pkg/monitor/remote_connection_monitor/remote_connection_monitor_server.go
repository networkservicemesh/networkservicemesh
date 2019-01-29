package remote_connection_monitor

import (
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

type RemoteConnectionMonitor struct {
	monitor.MonitorServer
}

func NewRemoteConnectionMonitor() *RemoteConnectionMonitor {
	rv := &RemoteConnectionMonitor{
		MonitorServer: monitor.NewMonitorServer(&RemoteConnectionEventConverter{}),
	}
	go rv.Serve()
	return rv
}

func (m *RemoteConnectionMonitor) MonitorConnections(selector *connection.MonitorScopeSelector, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	filtered := NewMonitorConnectionFilter(selector, recipient)
	return m.MonitorEntities(filtered)
}
