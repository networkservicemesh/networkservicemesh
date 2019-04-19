package crossconnect_monitor

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/metrics"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
	"github.com/sirupsen/logrus"
	"time"
)

type CrossConnectMonitor struct {
	monitor.MonitorServer
	statsCh chan *metrics.Statistics
}

func NewCrossConnectMonitor() *CrossConnectMonitor {
	rv := &CrossConnectMonitor{
		MonitorServer: monitor.NewMonitorServer(CreateCrossConnectEvent),
		statsCh:       make(chan *metrics.Statistics, 10),
	}
	go rv.Serve()
	//	go rv.serveMetrics()
	return rv
}

func (m *CrossConnectMonitor) serveMetrics() {
	for {
		select {
		case stat := <-m.statsCh:
			metrics := make(map[string]*crossconnect.Metrics)
			metrics[stat.Name] = &crossconnect.Metrics{
				Metrics: stat.Metrics,
			}
			logrus.Infof("New metrics (cross connect monitor): %v", stat)
			m.SendAll(CrossConnectEvent{
				EventImpl:  monitor.CrateEventImpl(monitor.UPDATE, m.Entities()),
				statistics: metrics,
			})
			logrus.Infof("Metrics sended (cross connect monitor): %v", stat)

			continue
		default:
			time.Sleep(time.Second)
		}
	}
}

func (m *CrossConnectMonitor) HandleMetrics(statistics *metrics.Statistics) {
	metrics := make(map[string]*crossconnect.Metrics)
	metrics[statistics.Name] = &crossconnect.Metrics{
		Metrics: statistics.Metrics,
	}
	logrus.Infof("New metrics (cross connect monitor): %v", statistics)
	m.SendAll(CrossConnectEvent{
		EventImpl:  monitor.CrateEventImpl(monitor.UPDATE, m.Entities()),
		statistics: metrics,
	})
	logrus.Infof("Metrics sended (cross connect monitor): %v", statistics)
}
func (m *CrossConnectMonitor) MonitorCrossConnects(_ *empty.Empty, recipient crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer) error {
	return m.MonitorEntities(recipient)
}
