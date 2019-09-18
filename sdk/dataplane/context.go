package dataplane

import (
	"context"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/sirupsen/logrus"

	monitor_crossconnect "github.com/networkservicemesh/networkservicemesh/controlplane/pkg/monitor/crossconnect"
	"github.com/networkservicemesh/networkservicemesh/dataplane/api/dataplane"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

type contextKeyType string

const (
	nextKey       contextKeyType = "nextKey"
	clientKey     contextKeyType = "clientKey"
	dataChangeKey contextKeyType = "dataChangeKey"
	loggerKey     contextKeyType = "loggerKey"
	monitorKey    contextKeyType = "monitorKey"
)

func withNext(ctx context.Context, handler dataplane.DataplaneServer) context.Context {
	return context.WithValue(ctx, nextKey, handler)
}

// Next returns next dataplane server of current chain state. Returns nil if context has not chain.
func Next(ctx context.Context) dataplane.DataplaneServer {
	if v, ok := ctx.Value(nextKey).(dataplane.DataplaneServer); ok {
		return v
	}
	return nil
}

// WithConfiguratorClient adds to context value with configurator client
func WithConfiguratorClient(ctx context.Context, endpoint string) (context.Context, func() error, error) {
	conn, err := tools.DialTCP(endpoint)
	if err != nil {
		Logger(ctx).Errorf("Can't dial grpc server: %v", err)
		return nil, nil, err
	}
	client := configurator.NewConfiguratorClient(conn)

	return context.WithValue(ctx, clientKey, client), conn.Close, nil
}

//ConfiguratorClient returns configurator client or nill if client not created
func ConfiguratorClient(ctx context.Context) configurator.ConfiguratorClient {
	if client, ok := ctx.Value(clientKey).(configurator.ConfiguratorClient); ok {
		return client
	}
	return nil
}

//WithDataChange puts dataChange config into context
func WithDataChange(ctx context.Context, dataChange *configurator.Config) context.Context {
	return context.WithValue(ctx, dataChangeKey, dataChange)
}

//DataChange gets dataChange config from context
func DataChange(ctx context.Context) *configurator.Config {
	if dataChange, ok := ctx.Value(dataChangeKey).(*configurator.Config); ok {
		return dataChange
	}
	return nil
}

//Logger returns logger from context
func Logger(ctx context.Context) logrus.FieldLogger {
	if logger, ok := ctx.Value(loggerKey).(logrus.FieldLogger); ok {
		return logger
	}
	return logrus.New()
}

//WithLogger puts logger into context
func WithLogger(ctx context.Context, logger logrus.FieldLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

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
