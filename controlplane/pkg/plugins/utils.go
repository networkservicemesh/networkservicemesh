package plugins

import (
	"context"
	"fmt"
	"net"
	"path"
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	registrationTimeout = 100 * time.Second
)

// StartPlugin creates an instance of a plugin and registers it
func StartPlugin(ctx context.Context, name, registry string, services map[plugins.PluginCapability]interface{}) error {
	span := spanhelper.FromContext(ctx, fmt.Sprintf("StartPlugin-%v", name))
	defer span.Finish()

	registryDir := path.Dir(registry) // create plugin's socket in the same directory as the registry
	endpoint := path.Join(registryDir, name+".sock")

	span.LogObject("registry", registry)
	span.LogObject("services", services)
	span.LogObject("endpoint", endpoint)
	span.LogObject("registry", registry)
	span.LogObject("registryDir", registryDir)

	if err := tools.SocketCleanup(endpoint); err != nil {
		span.LogError(err)
		return err
	}

	capabilities := make([]plugins.PluginCapability, 0, len(services))
	for capability := range services {
		capabilities = append(capabilities, capability)
	}
	span.LogObject("capabilities", capabilities)

	if err := createPlugin(span.Context(), name, endpoint, services); err != nil {
		span.LogError(err)
		return err
	}

	if err := registerPlugin(span.Context(), name, endpoint, registry, capabilities); err != nil {
		span.LogError(err)
		return err
	}

	return nil
}

func createPlugin(ctx context.Context, name, endpoint string, services map[plugins.PluginCapability]interface{}) error {
	sock, err := net.Listen("unix", endpoint)
	if err != nil {
		return err
	}

	server := tools.NewServer(ctx)

	for capability, service := range services {
		switch capability {
		case plugins.PluginCapability_CONNECTION:
			connectionService, ok := service.(plugins.ConnectionPluginServer)
			if !ok {
				return errors.New("the service cannot be used as a connection plugin since it does not implement ConnectionPluginServer interface")
			}
			plugins.RegisterConnectionPluginServer(server, connectionService)
		default:
			return errors.Errorf("unsupported capability: %v", capability)
		}
	}

	go func() {
		if err := server.Serve(sock); err != nil {
			logrus.Errorf("Failed to start a grpc server for '%s' plugin: %v", name, err)
		}
	}()

	return nil
}

func registerPlugin(ctx context.Context, name, endpoint, registry string, capabilities []plugins.PluginCapability) error {
	span := spanhelper.FromContext(ctx, "register-plugin")
	defer span.Finish()
	_ = tools.WaitForPortAvailable(span.Context(), "unix", registry, 100*time.Millisecond)
	conn, err := tools.DialContextUnix(span.Context(), registry)
	if err != nil {
		return errors.Wrap(err, "cannot connect to the Plugin Registry")
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			logrus.Fatalf("Cannot close the connection to the Plugin Registry: %v", err)
		}
	}()

	client := plugins.NewPluginRegistryClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), registrationTimeout)
	defer cancel()

	_, err = client.Register(ctx, &plugins.PluginInfo{
		Name:         name,
		Endpoint:     endpoint,
		Capabilities: capabilities,
	})

	return err
}
