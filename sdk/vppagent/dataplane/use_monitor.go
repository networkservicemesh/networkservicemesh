package dataplane

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/pkg/apis/dataplane"
)

//UseMonitor creates dataplane server handler with updating crossconnect monitor server
func UseMonitor(monitor monitor_crossconnect.MonitorServer) dataplane.DataplaneServer {
	return &useMonitor{
		monitor: monitor,
	}
}

type useMonitor struct {
	monitor monitor_crossconnect.MonitorServer
}

func (c *useMonitor) Request(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*crossconnect.CrossConnect, error) {
	if next := Next(ctx); next != nil {
		resp, err := next.Request(WithMonitor(ctx, c.monitor), crossConnect)
		if err == nil {
			c.monitor.Update(resp)
		}
		return resp, err
	}
	return crossConnect, nil
}

func (c *useMonitor) Close(ctx context.Context, crossConnect *crossconnect.CrossConnect) (*empty.Empty, error) {
	defer c.monitor.Delete(crossConnect)
	if next := Next(ctx); next != nil {
		return next.Close(ctx, crossConnect)
	}
	return new(empty.Empty), nil
}
