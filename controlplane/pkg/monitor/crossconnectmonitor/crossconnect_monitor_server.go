package crossconnectmonitor

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

// Server is a monitor.Server for crossconnect GRPC API
type Server struct {
	monitor.Server
	statsCh chan map[string]*crossconnect.Metrics
}

// NewServer creates a new Server
func NewServer() *Server {
	rv := &Server{
		Server:  monitor.NewServer(createEvent),
		statsCh: make(chan map[string]*crossconnect.Metrics, 10),
	}
	go rv.Serve()
	go rv.serveMetrics()
	return rv
}

func (m *Server) serveMetrics() {
	for {
		m.SendAll(event{
			EventImpl:  monitor.CrateEventImpl(monitor.UPDATE, m.Entities()),
			statistics: <-m.statsCh,
		})
	}
}

// HandleMetrics updates Server recipients with new metrics
func (m *Server) HandleMetrics(statistics map[string]*crossconnect.Metrics) {
	m.statsCh <- statistics
}

// MonitorCrossConnects adds recipient for Server events
func (m *Server) MonitorCrossConnects(_ *empty.Empty, recipient crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer) error {
	return m.MonitorEntities(recipient)
}
