package remote_connection_monitor

import "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/remote/connection"

type monitorConnectionFilter struct {
	connection.MonitorConnection_MonitorConnectionsServer

	selector *connection.MonitorScopeSelector
}

func NewMonitorConnectionFilter(selector *connection.MonitorScopeSelector, monitor connection.MonitorConnection_MonitorConnectionsServer) connection.MonitorConnection_MonitorConnectionsServer {
	return &monitorConnectionFilter{
		selector: selector,
		MonitorConnection_MonitorConnectionsServer: monitor,
	}
}

func (d *monitorConnectionFilter) Send(in *connection.ConnectionEvent) error {
	out := &connection.ConnectionEvent{
		Type:        in.Type,
		Connections: make(map[string]*connection.Connection),
	}
	for key, value := range in.GetConnections() {
		if value.GetSourceNetworkServiceManagerName() == d.selector.GetNetworkServiceManagerName() {
			out.Connections[key] = value
		}
		if value.GetDestinationNetworkServiceManagerName() == d.selector.GetNetworkServiceManagerName() {
			out.Connections[key] = value
		}
	}
	if len(out.Connections) > 0 || out.Type == connection.ConnectionEventType_INITIAL_STATE_TRANSFER {
		return d.MonitorConnection_MonitorConnectionsServer.Send(out)
	}
	return nil
}
