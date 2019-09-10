package dataplane

import (
	"context"

	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
)

const (
	monitorKey = "monitor"
)

func WithMonitor(ctx context.Context, monitor monitor_crossconnect.MonitorServer) context.Context {
	return context.WithValue(ctx, monitorKey, monitor)
}

func MonitorServer(ctx context.Context) monitor_crossconnect.MonitorServer {
	if monitor, ok := ctx.Value(monitorKey).(monitor_crossconnect.MonitorServer); ok {
		return monitor
	}
	return nil
}
