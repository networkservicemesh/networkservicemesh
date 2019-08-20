package plugins

import (
	"context"
	"fmt"
	"net"
	"path"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	registrationTimeout  = 100 * time.Second
	registrationInterval = 100 * time.Millisecond
)

// StartPlugin creates an instance of a plugin and registers it
func StartPlugin(name string, services map[plugins.PluginCapability]interface{}) error {
	endpoint := path.Join(plugins.PluginRegistryPath, name+".sock")
	if err := tools.SocketCleanup(endpoint); err != nil {
		return err
	}

	if err := createPlugin(name, endpoint, services); err != nil {
		return err
	}

	capabilities := make([]plugins.PluginCapability, 0, len(services))
	for capability := range services {
		capabilities = append(capabilities, capability)
	}

	pluginInfo := &plugins.PluginInfo{
		Name:         name,
		Endpoint:     endpoint,
		Capabilities: capabilities,
	}

	if err := registerPlugin(pluginInfo); err != nil {
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
		case plugins.PluginCapability_REQUEST:
			requestService, ok := service.(plugins.RequestPluginServer)
			if !ok {
				return fmt.Errorf("the service cannot be used as a request plugin since it does not implement RequestPluginServer interface")
			}
			plugins.RegisterRequestPluginServer(server, requestService)
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

func registerPlugin(pluginInfo *plugins.PluginInfo) error {
	err := tools.WaitForPortAvailable(context.Background(), "unix", plugins.PluginRegistrySocket, registrationInterval)
	if err != nil {
		return fmt.Errorf("cannot connect to the Plugin Registry: %v", err)
	}

	conn, err := tools.DialUnix(plugins.PluginRegistrySocket)
	if err != nil {
		return fmt.Errorf("cannot connect to the Plugin Registry: %v", err)
	}

	closeConn := func() {
		if err = conn.Close(); err != nil {
			logrus.Errorf("Cannot close the connection from '%s' plugin to the Plugin Registry: %v", pluginInfo.GetName(), err)
		}
	}

	client := plugins.NewPluginRegistryClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), registrationTimeout)
	defer cancel()

	if _, err = client.Register(ctx, pluginInfo); err != nil {
		closeConn()
		return err
	}

	go livenessMonitor(pluginInfo, client, closeConn)
	return nil
}

func livenessMonitor(pluginInfo *plugins.PluginInfo, client plugins.PluginRegistryClient, closeConn func()) { // TODO: add context?
	defer closeConn()

	stream, err := client.RequestLiveness(context.Background())
	if err != nil {
		logrus.Errorf("Failed to create liveness grpc channel with NSM for '%s' plugin: %v", pluginInfo.GetName(), err)
		return
	}

	for {
		if err := stream.RecvMsg(&empty.Empty{}); err != nil {
			logrus.Errorf("Failed to receive from liveness grpc channel with NSM of '%s' plugin: %v", pluginInfo.GetName(), err)

			go func() {
				if err = registerPlugin(pluginInfo); err != nil {
					logrus.Fatalf("Cannot re-register '%s' plugin in the Plugin Registry: %v", pluginInfo.GetName(), err)
				}
			}()

			return
		}
	}
}
