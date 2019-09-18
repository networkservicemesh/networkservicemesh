package remote

import "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"

type monitorConnectionFilter struct {
	connection.MonitorConnection_MonitorConnectionsServer

	selector *connection.MonitorScopeSelector
}

func newMonitorConnectionFilter(selector *connection.MonitorScopeSelector, monitor connection.MonitorConnection_MonitorConnectionsServer) connection.MonitorConnection_MonitorConnectionsServer {
	return &monitorConnectionFilter{
		selector: selector,
		MonitorConnection_MonitorConnectionsServer: monitor,
	}
}

// Send filters event connections and pass it to the next sending layer
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
