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
func StartPlugin(name string, capabilities []plugins.PluginCapability, service interface{}) error {
	endpoint := path.Join(plugins.PluginRegistryPath, name+".sock")
	if err := tools.SocketCleanup(endpoint); err != nil {
		return err
	}

	if err := createPlugin(name, endpoint, capabilities, service); err != nil {
		return err
	}

	if err := registerPlugin(name, endpoint, capabilities); err != nil {
		return err
	}

	return nil
}

func createPlugin(name string, endpoint string, capabilities []plugins.PluginCapability, service interface{}) error {
	sock, err := net.Listen("unix", endpoint)
	if err != nil {
		return err
	}

	server := tools.NewServer()

	for _, capability := range capabilities {
		switch capability {
		case plugins.PluginCapability_CONNECTION:
			connectionService, ok := service.(plugins.ConnectionPluginServer)
			if !ok {
				return fmt.Errorf("the service cannot be used as a connection plugin since it does not implement ConnectionPluginServer interface")
			}
			plugins.RegisterConnectionPluginServer(server, connectionService)
		}
	}

	go func() {
		if err := server.Serve(sock); err != nil {
			logrus.Errorf("Failed to start a grpc server for '%s' plugin: %v", name, err)
		}
	}()

	return nil
}

func registerPlugin(name string, endpoint string, capabilities []plugins.PluginCapability) error {
	conn, err := tools.DialUnix(plugins.PluginRegistrySocket)
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
