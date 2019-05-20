package crossconnect

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

// MonitorServer is a monitor.Server for crossconnect GRPC API
type MonitorServer struct {
	monitor.Server
	statsCh chan map[string]*crossconnect.Metrics
}

// NewMonitorServer creates a new MonitorServer
func NewMonitorServer() *MonitorServer {
	rv := &MonitorServer{
		Server:  monitor.NewServer(createEvent),
		statsCh: make(chan map[string]*crossconnect.Metrics, 10),
	}
	go rv.Serve()
	go rv.serveMetrics()
	return rv
}

func (m *MonitorServer) serveMetrics() {
	for {
		m.SendAll(event{
			EventImpl:  monitor.CrateEventImpl(monitor.UPDATE, m.Entities()),
			statistics: <-m.statsCh,
		})
	}
}

// HandleMetrics updates MonitorServer recipients with new metrics
func (m *MonitorServer) HandleMetrics(statistics map[string]*crossconnect.Metrics) {
	m.statsCh <- statistics
}

// MonitorCrossConnects adds recipient for MonitorServer events
func (m *MonitorServer) MonitorCrossConnects(_ *empty.Empty, recipient crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer) error {
	return m.MonitorEntities(recipient)
}
