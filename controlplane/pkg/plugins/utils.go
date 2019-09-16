package plugins

import (
	"context"
	"fmt"
	"net"
	"path"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	registrationTimeout = 100 * time.Second
)

// StartPlugin creates an instance of a plugin and registers it
func StartPlugin(name, registry string, services map[plugins.PluginCapability]interface{}) error {
	registryDir := path.Dir(registry) // create plugin's socket in the same directory as the registry
	endpoint := path.Join(registryDir, name+".sock")
	if err := tools.SocketCleanup(endpoint); err != nil {
		return err
	}

	capabilities := make([]plugins.PluginCapability, 0, len(services))
	for capability := range services {
		capabilities = append(capabilities, capability)
	}

	if err := createPlugin(name, endpoint, services); err != nil {
		return err
	}

	if err := registerPlugin(name, endpoint, registry, capabilities); err != nil {
		return err
	}

	return nil
}

func createPlugin(name, endpoint string, services map[plugins.PluginCapability]interface{}) error {
	sock, err := net.Listen("unix", endpoint)
	if err != nil {
		return err
	}

	server := tools.NewServer()

	for capability, service := range services {
		switch capability {
		case plugins.PluginCapability_CONNECTION:
			connectionService, ok := service.(plugins.ConnectionPluginServer)
			if !ok {
				return fmt.Errorf("the service cannot be used as a connection plugin since it does not implement ConnectionPluginServer interface")
			}
			plugins.RegisterConnectionPluginServer(server, connectionService)
		default:
			return fmt.Errorf("unsupported capability: %v", capability)
		}
	}

	go func() {
		if err := server.Serve(sock); err != nil {
			logrus.Errorf("Failed to start a grpc server for '%s' plugin: %v", name, err)
		}
	}()

	return nil
}

func registerPlugin(name, endpoint, registry string, capabilities []plugins.PluginCapability) error {
	_ = tools.WaitForPortAvailable(context.Background(), "unix", registry, 100*time.Millisecond)
	conn, err := tools.DialUnix(registry)
	if err != nil {
		return fmt.Errorf("cannot connect to the Plugin Registry: %v", err)
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
