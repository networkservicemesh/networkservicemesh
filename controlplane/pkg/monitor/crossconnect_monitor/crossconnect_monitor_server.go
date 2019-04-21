package crossconnect_monitor

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

type CrossConnectMonitor struct {
	monitor.MonitorServer
	statsCh chan map[string]*crossconnect.Metrics
}

func NewCrossConnectMonitor() *CrossConnectMonitor {
	rv := &CrossConnectMonitor{
		MonitorServer: monitor.NewMonitorServer(CreateCrossConnectEvent),
		statsCh:       make(chan map[string]*crossconnect.Metrics, 10),
	}
	go rv.Serve()
	go rv.serveMetrics()
	return rv
}

func (m *CrossConnectMonitor) serveMetrics() {
	for {
		select {
		case stat := <-m.statsCh:
			m.SendAll(CrossConnectEvent{
				EventImpl:  monitor.CrateEventImpl(monitor.UPDATE, m.Entities()),
				statistics: stat,
			})
		}
	}
}

func (m *CrossConnectMonitor) HandleMetrics(statistics map[string]*crossconnect.Metrics) {
	m.statsCh <- statistics
}
func (m *CrossConnectMonitor) MonitorCrossConnects(_ *empty.Empty, recipient crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer) error {
	return m.MonitorEntities(recipient)
}
