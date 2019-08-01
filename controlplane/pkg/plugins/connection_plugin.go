package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/nsm/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
)

// ConnectionPluginManager transmits each method call to all registered connection plugins
type ConnectionPluginManager interface {
	PluginManager
	UpdateConnection(context.Context, connection.Connection) error
	ValidateConnection(context.Context, connection.Connection) error
}

type connectionPluginManager struct {
	sync.RWMutex
	pluginClients []plugins.ConnectionPluginClient
}

func (cpm *connectionPluginManager) Register(conn *grpc.ClientConn) {
	client := plugins.NewConnectionPluginClient(conn)
	cpm.addClient(client)
}

func (cpm *connectionPluginManager) addClient(client plugins.ConnectionPluginClient) {
	cpm.Lock()
	defer cpm.Unlock()

	cpm.pluginClients = append(cpm.pluginClients, client)
}

func (cpm *connectionPluginManager) getClients() []plugins.ConnectionPluginClient {
	cpm.RLock()
	defer cpm.RUnlock()

	return cpm.pluginClients
}

func (cpm *connectionPluginManager) UpdateConnection(ctx context.Context, conn connection.Connection) error {
	connCtx := conn.GetContext()
	for _, plugin := range cpm.getClients() {
		ctx, cancel := context.WithTimeout(ctx, 100*time.Second)

		var err error
		connCtx, err = plugin.UpdateConnectionContext(ctx, connCtx)
		cancel()

		if err != nil {
			return fmt.Errorf("connection plugin returned an error: %v", err)
		}
	}
	conn.SetContext(connCtx)
	return nil
}

func (cpm *connectionPluginManager) ValidateConnection(ctx context.Context, conn connection.Connection) error {
	for _, plugin := range cpm.getClients() {
		ctx, cancel := context.WithTimeout(ctx, 100*time.Second)

		result, err := plugin.ValidateConnectionContext(ctx, conn.GetContext())
		cancel()

		if err != nil {
			return fmt.Errorf("connection plugin returned an error: %v", err)
		}

		if result.GetStatus() != plugins.ConnectionValidationStatus_SUCCESS {
			return fmt.Errorf(result.GetErrorMessage())
		}
	}
	return nil
}
