package local_connection_monitor

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

type LocalConnectionMonitor struct {
	monitor.MonitorServer
}

func NewLocalConnectionMonitor() *LocalConnectionMonitor {
	rv := &LocalConnectionMonitor{
		MonitorServer: monitor.NewMonitorServer(CreateLocalConnectionEvent),
	}
	go rv.Serve()
	return rv
}

func (m *LocalConnectionMonitor) MonitorConnections(_ *empty.Empty, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	return m.MonitorEntities(recipient)
}
