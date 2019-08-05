package plugins

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
)

// ConnectionPluginManager transmits each method call to all registered connection plugins
type ConnectionPluginManager interface {
	PluginManager
	UpdateConnectionContext(context.Context, *connectioncontext.ConnectionContext) (*connectioncontext.ConnectionContext, error)
	ValidateConnectionContext(context.Context, *connectioncontext.ConnectionContext) (*plugins.ConnectionValidationResult, error)
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

func (cpm *connectionPluginManager) UpdateConnectionContext(ctx context.Context, connCtx *connectioncontext.ConnectionContext) (*connectioncontext.ConnectionContext, error) {
	for _, plugin := range cpm.getClients() {
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		var err error
		connCtx, err = plugin.UpdateConnectionContext(pluginCtx, connCtx)
		cancel()

		if err != nil {
			return connCtx, fmt.Errorf("connection plugin returned an error: %v", err)
		}
	}
	return connCtx, nil
}

func (cpm *connectionPluginManager) ValidateConnectionContext(ctx context.Context, connCtx *connectioncontext.ConnectionContext) (*plugins.ConnectionValidationResult, error) {
	for _, plugin := range cpm.getClients() {
		pluginCtx, cancel := context.WithTimeout(ctx, pluginCallTimeout)

		result, err := plugin.ValidateConnectionContext(pluginCtx, connCtx)
		cancel()

		if err != nil {
			return result, fmt.Errorf("connection plugin returned an error: %v", err)
		}

		if result.GetStatus() != plugins.ConnectionValidationStatus_SUCCESS {
			return result, nil
		}
	}
	return &plugins.ConnectionValidationResult{Status: plugins.ConnectionValidationStatus_SUCCESS}, nil
}
