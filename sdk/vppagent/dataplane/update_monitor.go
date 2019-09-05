package dataplane

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
	"github.com/networkservicemesh/networkservicemesh/sdk/vppagent/dataplane/state"
)

func UpdateMonitor(monitor monitor_crossconnect.MonitorServer) dataplane.DataplaneServer {
	return &updateMonitor{
		monitor:                     monitor,
		EmptyChainedDataplaneServer: new(EmptyChainedDataplaneServer),
	}
}

type updateMonitor struct {
	*EmptyChainedDataplaneServer
	monitor monitor_crossconnect.MonitorServer
}

func (c *updateMonitor) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	c.monitor.Update(crossConnect)
	return state.NextDataplaneRequest(ctx, crossConnect)
}

func (c *updateMonitor) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	c.monitor.Delete(crossConnect)
	return state.NextDataplaneClose(ctx, crossConnect)
}
