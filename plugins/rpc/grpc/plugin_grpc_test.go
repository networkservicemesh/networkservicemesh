// Copyright (c) 2018 Cisco and/or its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package grpc_test

import (
	"errors"
	g "github.com/ligato/cn-infra/rpc/grpc"
	"github.com/ligato/networkservicemesh/plugins/rpc/grpc"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"testing"
)

const (
	// DefaultHelloWorldGreeting is the string prepended to names passed to the SayHello Service
	DefaultHelloWorldGreeting = "hello "
	DefaultName               = "Ed"
	DefaultResponse           = DefaultHelloWorldGreeting + DefaultName
)

// GreeterService implements GRPC GreeterServer interface (interface generated from protobuf definition file).
// It is a simple implementation for testing/demo only purposes.
type GreeterService struct{}

// SayHello returns error if request.name was not filled otherwise: "hello " + request.Name
func (*GreeterService) SayHello(ctx context.Context, request *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	if request.Name == "" {
		return nil, errors.New("not filled name in the request")
	}

	return &helloworld.HelloReply{Message: DefaultHelloWorldGreeting + request.Name}, nil
}

// Plugin creates a new type of Plugin for a particular GRPC Service by embedding grpc.Plugin
type Plugin struct {
	*grpc.Plugin
}

// Init overloads the grpc.Plugin Init() method
// It calls grpc.Init() on the embedded gRPC plugin and then register our GreeterService with the GRPC Server
func (plugin *Plugin) Init() (err error) {
	plugin.Log.Infof("Plugin Init()")
	err = plugin.Plugin.Init()
	if err != nil {
		return err
	}
	helloworld.RegisterGreeterServer(plugin.GetServer(), &GreeterService{})
	plugin.Log.Infof("Registered Greeter Service")
	return err
}

func NewPlugin(opts ...grpc.Option) *Plugin {
	return &Plugin{grpc.NewPlugin(opts...)}
}

func TestPlugin(t *testing.T) {
	RegisterTestingT(t)
	plugin := NewPlugin(grpc.UseConf(g.Config{Endpoint: "localhost:1234"})) // Need to provide Config
	err := plugin.Init()
	Expect(err).To(BeNil())
}
