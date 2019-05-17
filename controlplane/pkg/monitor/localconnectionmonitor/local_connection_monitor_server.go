package localconnectionmonitor

import (
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

// Server is a monitor.Server for localconnection GRPC API
type Server struct {
	monitor.Server
}

// NewServer creates a new Server
func NewServer() *Server {
	rv := &Server{
		Server: monitor.NewServer(createEvent),
	}
	go rv.Serve()
	return rv
}

// MonitorConnections adds recipient for Server events
func (m *Server) MonitorConnections(_ *empty.Empty, recipient connection.MonitorConnection_MonitorConnectionsServer) error {
	return m.MonitorEntities(recipient)
}
