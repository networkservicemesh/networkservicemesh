package dataplane

import (
	"context"

	"github.com/ligato/vpp-agent/api/configurator"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

type clientKeyType string

const (
	clientKey clientKeyType = "client"
)

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
