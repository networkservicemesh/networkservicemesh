package plugins

import (
	"context"
	"net"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// PluginRegistry stores a plugin manager for each plugin type
type PluginRegistry interface {
	Start() error
	Stop() error

	GetConnectionPluginManager() ConnectionPluginManager
}

type pluginRegistry struct {
	sync.RWMutex
	connections             []*grpc.ClientConn
	connectionPluginManager ConnectionPluginManager
}

type pluginManager interface {
	register(*grpc.ClientConn)
}

// NewPluginRegistry creates an instance of PluginRegistry
func NewPluginRegistry() PluginRegistry {
	return &pluginRegistry{
		connectionPluginManager: &connectionPluginManager{},
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

	tracer := opentracing.GlobalTracer()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.StreamInterceptor(
			otgrpc.OpenTracingStreamServerInterceptor(tracer)))

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
	conn, err := pr.createConnection(info.Endpoint)
	if err != nil {
		return nil, err
	}
	for _, feature := range info.Features {
		if feature == plugins.PluginType_CONNECTION {
			pr.connectionPluginManager.register(conn)
		}
	}
	return &empty.Empty{}, nil
}

func (pr *pluginRegistry) createConnection(endpoint string) (*grpc.ClientConn, error) {
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial("unix:"+endpoint, grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)))
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
