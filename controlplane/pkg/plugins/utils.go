package plugins

import (
	"context"
	"fmt"
	"net"
	"path"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	registrationTimeout = 100 * time.Second
)

type registrationMonitor struct {
	clientName     string
	livenessStream grpc.ClientStream
	ctx            context.Context
	register       func() error
	onDone         func()
}

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

	regmon := &registrationMonitor{
		clientName: name,
		ctx:        context.Background(),
	}
	regmon.register = func() error { return regmon.registerPlugin(name, endpoint, registry, capabilities) }

	return regmon.register()
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

func (rm *registrationMonitor) registerPlugin(name, endpoint, registry string, capabilities []plugins.PluginCapability) error {
	_ = tools.WaitForPortAvailable(context.Background(), "unix", registry, 100*time.Millisecond)
	conn, err := tools.DialUnix(registry)
	if err != nil {
		return fmt.Errorf("cannot connect to the Plugin Registry: %v", err)
	}

	cleanup := func() {
		err = conn.Close()
		if err != nil {
			logrus.Fatalf("Cannot close the connection to the Plugin Registry: %v", err)
		}
	}
	defer func() {
		if err != nil {
			cleanup()
		} else {
			rm.onDone = cleanup
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), registrationTimeout)
	defer cancel()

	client := plugins.NewPluginRegistryClient(conn)
	_, err = client.Register(ctx, &plugins.PluginInfo{
		Name:         name,
		Endpoint:     endpoint,
		Capabilities: capabilities,
	})
	if err != nil {
		logrus.Errorf("%s: failed to register plugin: %s, grpc code: %+v", name, err, status.Convert(err).Code())
		return err
	}

	logrus.Infof("%s: starting NSM liveness monitoring", rm.clientName)
	rm.livenessStream, err = client.RequestLiveness(context.Background())
	if err != nil {
		logrus.Errorf("%s: failed to get NSM liveness channel: %s, grpc code: %+v", name, err, status.Convert(err).Code())
		return err
	}

	go rm.monitor()

	return err
}

func (rm *registrationMonitor) monitor() {
	for {
		select {
		case <-rm.ctx.Done():
			logrus.Infof("%s: finishing NSM liveness monitoring", rm.clientName)
			if rm.onDone != nil {
				rm.onDone()
			}
			return
		default:
			err := rm.livenessStream.RecvMsg(&empty.Empty{})
			if err != nil {
				logrus.Errorf("%s: failed to read liveness channel: %s, grpc code: %+v",
					rm.clientName, err, status.Convert(err).Code())
				logrus.Infof("%s: starting re-registration procedure...", rm.clientName)
				if rm.register() == nil {
					logrus.Infof("%s: successfully re-registered plugin", rm.clientName)
				}
				return
			}
		}
	}
}
