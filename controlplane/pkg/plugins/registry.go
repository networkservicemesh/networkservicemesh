package plugins

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	registryInitSleep   = 1 * time.Second
	registryInitTimeout = 10 * time.Second
	pluginCallTimeout   = 100 * time.Second
)

const (
	hasDiscoveryPlugin = 1 << iota
	hasRegistryPlugin

	hasAllRequiredPlugins       = hasDiscoveryPlugin | hasRegistryPlugin
	requiredPluginsErrorMessage = "timeout waiting for required plugins, need at least one discovery and one registry plugin"
)

// PluginRegistry stores a plugin manager for each plugin type
type PluginRegistry interface {
	Start() error
	Stop() error

	GetConnectionPluginManager() ConnectionPluginManager
	GetDiscoveryPluginManager() DiscoveryPluginManager
	GetRegistryPluginManager() RegistryPluginManager
}

// PluginManager allows to register a client connection
type PluginManager interface {
	Register(string, *grpc.ClientConn)
}

type pluginRegistry struct {
	sync.RWMutex
	status                  int
	connections             sync.Map
	connectionPluginManager ConnectionPluginManager
	discoveryPluginManager  DiscoveryPluginManager
	registryPluginManager   RegistryPluginManager
}

// NewPluginRegistry creates an instance of PluginRegistry
func NewPluginRegistry() PluginRegistry {
	return &pluginRegistry{
		status:                  hasAllRequiredPlugins, // TODO: remove this when discovery and registry plugins will be implemented
		connections:             sync.Map{},
		connectionPluginManager: createConnectionPluginManager(),
		discoveryPluginManager:  createDiscoveryPluginManager(),
		registryPluginManager:   createRegistryPluginManager(),
	}
}

func (pr *pluginRegistry) Start() error {
	if err := tools.SocketCleanup(plugins.PluginRegistrySocket); err != nil {
		return err
	}

	sock, err := net.Listen("unix", plugins.PluginRegistrySocket)
	if err != nil {
		return err
	}

	server := tools.NewServer()
	plugins.RegisterPluginRegistryServer(server, pr)

	go func() {
		if err := server.Serve(sock); err != nil {
			logrus.Fatalf("Failed to serve: %v", err)
		}
	}()

	return pr.waitForRequiredPlugins()
}

func (pr *pluginRegistry) waitForRequiredPlugins() error {
	st := time.Now()
	for {
		if (pr.getStatus() & hasAllRequiredPlugins) != 0 {
			break
		}
		if time.Since(st) > registryInitTimeout {
			return fmt.Errorf(requiredPluginsErrorMessage)
		}
		time.Sleep(registryInitSleep)
	}
	return nil
}

func (pr *pluginRegistry) Stop() error {
	var rv error
	pr.connections.Range(func(key interface{}, value interface{}) bool {
		name := key.(string)
		conn := value.(*grpc.ClientConn)

		if err := conn.Close(); err != nil {
			rv = fmt.Errorf("failed to close connection to '%s' plugin: %v", name, err)
			return false
		}
		return true
	})
	return rv
}

func (pr *pluginRegistry) Register(ctx context.Context, info *plugins.PluginInfo) (*empty.Empty, error) {
	if info.GetName() == "" || info.GetEndpoint() == "" || len(info.Capabilities) == 0 {
		return nil, fmt.Errorf("invalid registration data, expected non-empty name, endpoint and capabilities list")
	}
	logrus.Infof("Registering a plugin: name '%s', endpoint '%s', capabilities %v", info.GetName(), info.GetEndpoint(), info.GetCapabilities())

	conn, err := pr.createConnection(info.GetName(), info.GetEndpoint())
	if err != nil {
		return nil, err
	}

	for _, capability := range info.GetCapabilities() {
		switch capability {
		case plugins.PluginCapability_CONNECTION:
			pr.connectionPluginManager.Register(info.GetName(), conn)
		case plugins.PluginCapability_DISCOVERY:
			pr.discoveryPluginManager.Register(info.GetName(), conn)
			pr.addStatus(hasDiscoveryPlugin)
		case plugins.PluginCapability_REGISTRY:
			pr.registryPluginManager.Register(info.GetName(), conn)
			pr.addStatus(hasRegistryPlugin)
		default:
			return nil, fmt.Errorf("unsupported capability: %v", capability)
		}
	}

	return &empty.Empty{}, nil
}

func (pr *pluginRegistry) addStatus(status int) {
	pr.Lock()
	defer pr.Unlock()

	pr.status |= status
}

func (pr *pluginRegistry) getStatus() int {
	pr.RLock()
	defer pr.RUnlock()

	return pr.status
}

func (pr *pluginRegistry) createConnection(name, endpoint string) (*grpc.ClientConn, error) {
	conn, err := tools.DialUnix(endpoint)
	if err != nil {
		return nil, err
	}

	if _, ok := pr.connections.Load(name); ok {
		return nil, fmt.Errorf("already have a plugin with the same name")
	}

	pr.connections.Store(name, conn)
	return conn, err
}

func (pr *pluginRegistry) GetConnectionPluginManager() ConnectionPluginManager {
	return pr.connectionPluginManager
}

func (pr *pluginRegistry) GetDiscoveryPluginManager() DiscoveryPluginManager {
	return pr.discoveryPluginManager
}

func (pr *pluginRegistry) GetRegistryPluginManager() RegistryPluginManager {
	return pr.registryPluginManager
}
