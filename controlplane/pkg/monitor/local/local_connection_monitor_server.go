package local

import (
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

// MonitorServer is a monitor.Server for local/connection GRPC API
type MonitorServer struct {
	monitor.Server
}

// NewMonitorServer creates a new MonitorServer
func NewMonitorServer() *MonitorServer {
	rv := &MonitorServer{
		Server: monitor.NewServer(createEvent),
	}
	go rv.Serve()
	return rv
}

// MonitorConnections adds recipient for MonitorServer events
func (m *MonitorServer) MonitorConnections(_ *empty.Empty, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	return m.MonitorEntities(recipient)
}
