package dataplane

import (
	"context"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	clientKey = "client"
)

func WithConfiguratorClient(ctx context.Context, endpoint string) (context.Context, func() error, error) {
	conn, err := tools.DialTCP(endpoint)
	if err != nil {
		logrus.Errorf("Can't dial grpc server: %v", err)
		return nil, nil, err
	}
	client := configurator.NewConfiguratorClient(conn)

	return context.WithValue(ctx, clientKey, client), conn.Close, nil
}

func ConfigurationClient(ctx context.Context) configurator.ConfiguratorClient {
	if client, ok := ctx.Value(clientKey).(configurator.ConfiguratorClient); ok {
		return client
	}
	return nil
}
