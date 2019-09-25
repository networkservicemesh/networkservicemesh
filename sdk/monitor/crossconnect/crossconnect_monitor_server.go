package crossconnect

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor"
	"github.com/networkservicemesh/networkservicemesh/sdk/monitor/metrics"
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
		Server:  monitor.NewServer(&eventFactory{}),
		statsCh: make(chan map[string]*crossconnect.Metrics, 10),
	}
	go rv.Serve()
	go rv.serveMetrics()
	return rv
}

func (s *monitorServer) serveMetrics() {
	for {
		s.SendAll(&Event{
			BaseEvent:  monitor.NewBaseEvent(context.Background(), monitor.EventTypeUpdate, s.Entities()),
			Statistics: <-s.statsCh,
		})
	}
}

// HandleMetrics updates MonitorServer recipients with new metrics
func (s *monitorServer) HandleMetrics(statistics map[string]*crossconnect.Metrics) {
	s.statsCh <- statistics
}

// MonitorCrossConnects adds recipient for MonitorServer events
func (s *monitorServer) MonitorCrossConnects(_ *empty.Empty, recipient crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer) error {
	s.MonitorEntities(recipient)
	return nil
}
