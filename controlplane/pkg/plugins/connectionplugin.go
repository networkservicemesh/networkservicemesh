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

type ConnectionPluginManager interface {
	pluginManager
	UpdateConnection(connection.Connection)
	ValidateConnection(connection.Connection) error
}

type connectionPluginManager struct {
	pluginClients []plugins.ConnectionPluginClient
}

func (cpm *connectionPluginManager) register(conn *grpc.ClientConn) {
	client := plugins.NewConnectionPluginClient(conn)
	cpm.pluginClients = append(cpm.pluginClients, client)
}

func (cpm *connectionPluginManager) UpdateConnection(conn connection.Connection) {
	connCtx := conn.GetContext()
	for _, plugin := range cpm.pluginClients {
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

func (cpm *connectionPluginManager) ValidateConnection(conn connection.Connection) error {
	for _, plugin := range cpm.pluginClients {
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
