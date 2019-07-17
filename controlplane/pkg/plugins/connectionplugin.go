package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type ConnectionPluginRegistry interface {
	pluginManager
	UpdateConnection(connection.Connection)
	ValidateConnection(connection.Connection) error
}

type connectionPluginRegistry struct {
	connectionPlugins []plugins.ConnectionPluginClient
}

func (cpr *connectionPluginRegistry) registerPlugin(conn *grpc.ClientConn) {
	client := plugins.NewConnectionPluginClient(conn)
	cpr.connectionPlugins = append(cpr.connectionPlugins, client)
}

func (cpr *connectionPluginRegistry) UpdateConnection(conn connection.Connection) {
	connCtx := conn.GetContext()
	for _, plugin := range cpr.connectionPlugins {
		ctx, cancel := context.WithTimeout(context.Background(), 100 * time.Second)

		var err error
		connCtx, err = plugin.UpdateConnectionContext(ctx, connCtx)
		cancel()

		if err != nil {
			logrus.Errorf("Connection Plugin returned an error: %v", err)
		}
	}
	conn.SetContext(connCtx)
}

func (cpr *connectionPluginRegistry) ValidateConnection(conn connection.Connection) error {
	for _, plugin := range cpr.connectionPlugins {
		ctx, cancel := context.WithTimeout(context.Background(), 100 * time.Second)

		result, err := plugin.ValidateConnectionContext(ctx, conn.GetContext())
		cancel()

		if err != nil {
			logrus.Errorf("Connection Plugin returned an error: %v", err)
			continue
		}

		if result.GetStatus() != plugins.ConnectionValidationStatus_SUCCESS {
			return fmt.Errorf(result.GetErrorMessage())
		}
	}
	return nil
}
