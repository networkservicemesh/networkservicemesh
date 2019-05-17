package crossconnect

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

// MonitorServer is a monitor.Server for crossconnect GRPC API
type MonitorServer interface {
	monitor.Server
	metrics.MetricsMonitor
	crossconnect.MonitorCrossConnectServer
}

type monitorServer struct {
	monitor.Server
	statsCh chan map[string]*crossconnect.Metrics
}

// NewMonitorServer creates a new MonitorServer
func NewMonitorServer() MonitorServer {
	rv := &monitorServer{
		Server:  monitor.NewServer(createEvent),
		statsCh: make(chan map[string]*crossconnect.Metrics, 10),
	}
	go rv.Serve()
	go rv.serveMetrics()
	return rv
}

func (m *monitorServer) serveMetrics() {
	for {
		m.SendAll(event{
			EventImpl:  monitor.CrateEventImpl(monitor.UPDATE, m.Entities()),
			statistics: <-m.statsCh,
		})
	}
}

// HandleMetrics updates MonitorServer recipients with new metrics
func (m *monitorServer) HandleMetrics(statistics map[string]*crossconnect.Metrics) {
	m.statsCh <- statistics
}

// MonitorCrossConnects adds recipient for MonitorServer events
func (m *monitorServer) MonitorCrossConnects(_ *empty.Empty, recipient crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer) error {
	return m.MonitorEntities(recipient)
}
