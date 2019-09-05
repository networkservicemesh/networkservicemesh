package state

import (
	"context"
	"fmt"

	"github.com/ligato/vpp-agent/api/configurator"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

type connectionClientState string

const (
	clientKey connectionClientState = "client"
)

type closeableClient struct {
	client    configurator.ConfiguratorClient
	closeFunc func() error
}

func WithConfiguratorClient(ctx context.Context, endpoint string) (context.Context, error) {
	conn, err := tools.DialTCP(endpoint)
	if err != nil {
		logrus.Errorf("Can't dial grpc server: %v", err)
		return nil, err
	}
	client := configurator.NewConfiguratorClient(conn)

	return context.WithValue(ctx, clientKey, &closeableClient{
		client:    client,
		closeFunc: conn.Close,
	}), nil
}

func ConfigurationClient(ctx context.Context) configurator.ConfiguratorClient {
	rawClient := ctx.Value(clientKey)
	if rawClient == nil {
		return nil
	}
	if client, ok := rawClient.(*closeableClient); ok {
		return client.client
	}
	return nil
}

func CloseConnection(ctx context.Context) error {
	rawClient := ctx.Value(clientKey)
	if rawClient == nil {
		return nil
	}

	if client, ok := rawClient.(*closeableClient); ok {
		return client.closeFunc()
	}

	return fmt.Errorf("can not cast to cleint: %v", rawClient)
}
