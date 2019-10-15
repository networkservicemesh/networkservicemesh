package compat

import (
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	remote "github.com/networkservicemesh/networkservicemesh/controlplane/api/remote/connection"
)

type monitorConnectionServerAdapter struct {
	remoteServer remote.MonitorConnectionServer
	localServer  local.MonitorConnectionServer
}

func NewMonitorConnectionServerAdapter(remoteServer remote.MonitorConnectionServer, localServer local.MonitorConnectionServer) connection.MonitorConnectionServer {
	return &monitorConnectionServerAdapter{
		remoteServer: remoteServer,
		localServer:  localServer,
	}
}

type localMonitorConnection_MonitorConnectionsServerAdapter struct {
	connection.MonitorConnection_MonitorConnectionsServer
}

func (l localMonitorConnection_MonitorConnectionsServerAdapter) Send(event *local.ConnectionEvent) error {
	return l.MonitorConnection_MonitorConnectionsServer.Send(ConnectionEventLocalToUnified(event))
}

func (l localMonitorConnection_MonitorConnectionsServerAdapter) SendMsg(m interface{}) error {
	if e, ok := m.(*local.ConnectionEvent); ok {
		return l.Send(e)
	}
	return l.MonitorConnection_MonitorConnectionsServer.SendMsg(m)
}

func NewLocalMonitorConnection_MonitorConnectionsServerAdapter(srv connection.MonitorConnection_MonitorConnectionsServer) local.MonitorConnection_MonitorConnectionsServer {
	return &localMonitorConnection_MonitorConnectionsServerAdapter{
		MonitorConnection_MonitorConnectionsServer: srv,
	}
}

type remoteMonitorConnection_MonitorConnectionsServerAdapter struct {
	connection.MonitorConnection_MonitorConnectionsServer
}

func (r remoteMonitorConnection_MonitorConnectionsServerAdapter) Send(event *remote.ConnectionEvent) error {
	return r.MonitorConnection_MonitorConnectionsServer.Send(ConnectionEventRemoteToUnified(event))
}

func (r remoteMonitorConnection_MonitorConnectionsServerAdapter) SendMsg(m interface{}) error {
	if e, ok := m.(*remote.ConnectionEvent); ok {
		return r.Send(e)
	}
	return r.MonitorConnection_MonitorConnectionsServer.SendMsg(m)
}

func NewRemoteMonitorConnection_MonitorConnectionsServerAdapter(srv connection.MonitorConnection_MonitorConnectionsServer) remote.MonitorConnection_MonitorConnectionsServer {
	return &remoteMonitorConnection_MonitorConnectionsServerAdapter{
		MonitorConnection_MonitorConnectionsServer: srv,
	}
}

func (m monitorConnectionServerAdapter) MonitorConnections(selector *connection.MonitorScopeSelector, srv connection.MonitorConnection_MonitorConnectionsServer) error {
	// TODO Error handling
	if m.remoteServer != nil {
		m.remoteServer.MonitorConnections(MonitorScopeSelectorUnifiedToRemote(selector), NewRemoteMonitorConnection_MonitorConnectionsServerAdapter(srv))
	}
	if m.localServer != nil {
		m.localServer.MonitorConnections(&empty.Empty{}, NewLocalMonitorConnection_MonitorConnectionsServerAdapter(srv))
	}
	return nil
}
