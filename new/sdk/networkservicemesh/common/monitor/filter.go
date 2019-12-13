package monitor

import "github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"

type monitorFilter struct {
	selector *connection.MonitorScopeSelector
	connection.MonitorConnection_MonitorConnectionsServer
}

func newMonitorFilter(selector *connection.MonitorScopeSelector, srv connection.MonitorConnection_MonitorConnectionsServer) *monitorFilter {
	return &monitorFilter{
		selector: selector,
		MonitorConnection_MonitorConnectionsServer: srv,
	}
}

func (m *monitorFilter) Send(event *connection.ConnectionEvent) error {
	rv := &connection.ConnectionEvent{
		Type:        event.Type,
		Connections: connection.FilterMapOnManagerScopeSelector(event.GetConnections(), m.selector),
	}
	if rv.Type == connection.ConnectionEventType_INITIAL_STATE_TRANSFER || len(rv.GetConnections()) > 0 {
		return m.MonitorConnection_MonitorConnectionsServer.Send(rv)
	}
	return nil
}
