package dataplane

import (
	"context"

	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
)

type monitorKeyType string

const (
	monitorKey monitorKeyType = "monitor"
)

//WithMonitor puts into context cross connect monitor server
func WithMonitor(ctx context.Context, monitor monitor_crossconnect.MonitorServer) context.Context {
	return context.WithValue(ctx, monitorKey, monitor)
}

//MonitorServer gets from context cross connect monitor server
func MonitorServer(ctx context.Context) monitor_crossconnect.MonitorServer {
	if monitor, ok := ctx.Value(monitorKey).(monitor_crossconnect.MonitorServer); ok {
		return monitor
	}
	return nil
}
