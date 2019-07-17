package plugins

import (
	"context"
	"net"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/plugins"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type PluginRegistry interface {
	Start() error
	Stop() error

	GetConnectionPluginRegistry() ConnectionPluginRegistry
}

type pluginRegistry struct {
	connectionPluginRegistry ConnectionPluginRegistry
}

type pluginManager interface { // TODO: improve naming
	registerPlugin(string) error
}

func NewPluginRegistry() PluginRegistry {
	return &pluginRegistry{
		connectionPluginRegistry: &connectionPluginRegistry{},
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
			logrus.Fatalf("failed to serve: %v", err)
		}
	}()

	return nil
}

func (pr *pluginRegistry) Stop() error {
	// TODO: close all client connections
	return nil
}

func (pr *pluginRegistry) Register(ctx context.Context, info *plugins.PluginInfo) (*empty.Empty, error) {
	err := pr.connectionPluginRegistry.registerPlugin(info.Endpoint) // TODO: implement capabilities
	return &empty.Empty{}, err
}

func (pr *pluginRegistry) GetConnectionPluginRegistry() ConnectionPluginRegistry {
	return pr.connectionPluginRegistry
}
