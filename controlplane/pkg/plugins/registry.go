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
	pluginCallTimeout = 100 * time.Second
)

// PluginRegistry stores a plugin manager for each plugin type
type PluginRegistry interface {
	Start() error
	Stop() error

	GetConnectionPluginManager() ConnectionPluginManager
}

// PluginManager allows to register a client connection
type PluginManager interface {
	Register(string, *grpc.ClientConn) error
}

type pluginRegistry struct {
	sync.RWMutex
	connections             []*grpc.ClientConn
	connectionPluginManager ConnectionPluginManager
}

// NewPluginRegistry creates an instance of PluginRegistry
func NewPluginRegistry() PluginRegistry {
	return &pluginRegistry{
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
	for _, conn := range pr.getConnections() {
		err := conn.Close()
		if err != nil {
			logrus.Errorf("Failed to close connection: %v", err)
		}
	}
	return nil
}

func (pr *pluginRegistry) Register(ctx context.Context, info *plugins.PluginInfo) (*empty.Empty, error) {
	if info.GetName() == "" || info.GetEndpoint() == "" || len(info.Capabilities) == 0 {
		return nil, fmt.Errorf("invalid registration data, expected non-empty name, endpoint and capabilities list")
	}
	logrus.Infof("Registering a plugin: name '%s', endpoint '%s', capabilities %v", info.GetName(), info.GetEndpoint(), info.GetCapabilities())

	conn, err := pr.createConnection(info.GetEndpoint())
	if err != nil {
		return nil, err
	}
	for _, capability := range info.GetCapabilities() {
		switch capability {
		case plugins.PluginCapability_CONNECTION:
			if err := pr.connectionPluginManager.Register(info.GetName(), conn); err != nil {
				return nil, err
			}
		}
	}
	return &empty.Empty{}, nil
}

func (pr *pluginRegistry) createConnection(endpoint string) (*grpc.ClientConn, error) {
	conn, err := tools.DialUnix(endpoint)
	if err != nil {
		return nil, err
	}

	pr.addConnection(conn)
	return conn, nil
}

func (pr *pluginRegistry) addConnection(conn *grpc.ClientConn) {
	pr.Lock()
	defer pr.Unlock()

	pr.connections = append(pr.connections, conn)
}

func (pr *pluginRegistry) getConnections() []*grpc.ClientConn {
	pr.RLock()
	defer pr.RUnlock()

	return pr.connections
}

func (pr *pluginRegistry) GetConnectionPluginManager() ConnectionPluginManager {
	return pr.connectionPluginManager
}
