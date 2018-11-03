package main

import (
	"errors"

	"golang.org/x/net/context"
	"google.golang.org/grpc/examples/helloworld/helloworld"

	"github.com/ligato/cn-infra/core"
	"github.com/ligato/cn-infra/flavors/local"
	"github.com/ligato/cn-infra/flavors/rpc"
	"github.com/ligato/cn-infra/rpc/grpc"
)

// *************************************************************************
// This file contains GRPC service exposure example. To register service use
// Server.RegisterService(descriptor, service)
// ************************************************************************/

func main() {
	// Init close channel to stop the example after everything was logged
	exampleFinished := make(chan struct{}, 1)

	// Start Agent with ExamplePlugin & FlavorRPC (reused cn-infra plugins).
	agent := rpc.NewAgent(rpc.WithPlugins(func(flavor *rpc.FlavorRPC) []*core.NamedPlugin {

		examplePlug := &ExamplePlugin{exampleFinished: exampleFinished, Deps: Deps{
			PluginLogDeps: *flavor.LogDeps("example"),
			GRPC:          &flavor.GRPC,
		}}

		return []*core.NamedPlugin{{examplePlug.PluginName, examplePlug}}
	}))
	core.EventLoopWithInterrupt(agent, exampleFinished)
}

// ExamplePlugin presents the PluginLogger API.
type ExamplePlugin struct {
	Deps
	exampleFinished chan struct{}
}

// Deps - dependencies for ExamplePlugin
type Deps struct {
	local.PluginLogDeps
	GRPC grpc.Server
}

// Init demonstrates the usage of PluginLogger API.
func (plugin *ExamplePlugin) Init() (err error) {
	plugin.Log.Info("Example Init")

	helloworld.RegisterGreeterServer(plugin.GRPC.GetServer(), &GreeterService{})

	return nil
}

// GreeterService implements GRPC GreeterServer interface (interface generated from protobuf definition file).
// It is a simple implementation for testing/demo only purposes.
type GreeterService struct{}

// SayHello returns error if request.name was not filled otherwise: "hello " + request.Name
func (*GreeterService) SayHello(ctx context.Context, request *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	if request.Name == "" {
		return nil, errors.New("not filled name in the request")
	}

	return &helloworld.HelloReply{Message: "hello " + request.Name}, nil
}
