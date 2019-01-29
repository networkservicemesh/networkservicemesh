package crossconnect_monitor

import (
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor"
)

type CrossConnectMonitor struct {
	monitor.MonitorServer
}

func NewCrossConnectMonitor() *CrossConnectMonitor {
	rv := &CrossConnectMonitor{
		MonitorServer: monitor.NewMonitorServer(&CrossConnectEventConverter{}),
	}
	go rv.Serve()
	return rv
}

func (m *CrossConnectMonitor) MonitorCrossConnects(_ *empty.Empty, recipient crossconnect.MonitorCrossConnect_MonitorCrossConnectsServer) error {
	return m.MonitorEntities(recipient)
}
