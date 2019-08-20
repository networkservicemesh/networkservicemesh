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
	livenessInterval  = 5 * time.Second
	pluginCallTimeout = 100 * time.Second
)

// PluginRegistry stores a plugin manager for each plugin type
type PluginRegistry interface {
	Start() error
	Stop() error

	GetRequestPluginManager() RequestPluginManager
	GetConnectionPluginManager() ConnectionPluginManager
}

// PluginManager allows to register a client connection
type PluginManager interface {
	Register(string, *grpc.ClientConn)
}

type pluginRegistry struct {
	connections             sync.Map
	requestPluginManager    RequestPluginManager
	connectionPluginManager ConnectionPluginManager
}

// NewPluginRegistry creates an instance of PluginRegistry
func NewPluginRegistry() PluginRegistry {
	return &pluginRegistry{
		connections:             sync.Map{},
		requestPluginManager:    createRequestPluginManager(),
		connectionPluginManager: createConnectionPluginManager(),
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
		case plugins.PluginCapability_REQUEST:
			pr.requestPluginManager.Register(info.GetName(), conn)
		case plugins.PluginCapability_CONNECTION:
			pr.connectionPluginManager.Register(info.GetName(), conn)
		default:
			return nil, fmt.Errorf("unsupported capability: %v", capability)
		}
	}

	return &empty.Empty{}, nil
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

func (pr *pluginRegistry) RequestLiveness(liveness plugins.PluginRegistry_RequestLivenessServer) error {
	for {
		if err := liveness.SendMsg(&empty.Empty{}); err != nil {
			return err
		}
		time.Sleep(livenessInterval)
	}
}

func (pr *pluginRegistry) GetRequestPluginManager() RequestPluginManager {
	return pr.requestPluginManager
}

func (pr *pluginRegistry) GetConnectionPluginManager() ConnectionPluginManager {
	return pr.connectionPluginManager
}
