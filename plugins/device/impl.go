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

package device

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/networkservicemesh/utils/helper/plugintools"
	"github.com/ligato/networkservicemesh/utils/idempotent"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

// Plugin - provides machinery for device plugins
type Plugin struct {
	idempotent.Impl
	Deps
	*Config
	grpcServer *grpc.Server
}

// Init initializes ObjectStore plugin
func (p *Plugin) Init() error {
	return p.IdempotentInit(plugintools.LoggingInitFunc(p.Log, p, p.init))
}

func (p *Plugin) init() error {
	conf, ok := p.Deps.ConfigLoader.LoadConfig().(*Config)
	if !ok {
		return fmt.Errorf("type of config returned from config loader did not match p.Config")
	}
	p.Config = conf
	err := p.validateConfig()
	if err != nil {
		return err
	}
	err = p.startDeviceServer()
	if err != nil {
		return err
	}
	return p.register()
}

func (p *Plugin) validateConfig() error {
	if p.Config.KubeletSocket == "" {
		p.Config.KubeletSocket = pluginapi.KubeletSocket
	}
	if p.Config.DevicePath == "" {
		return fmt.Errorf("failed to set Plugin.Config.DevicePath, may not be empty string")
	}
	if p.Config.ServerSock == "" {
		return fmt.Errorf("failed to set Plugin.Config.ServerSock, may not be empty string")
	}
	if p.Config.ResourceName == "" {
		return fmt.Errorf("failed to set Plugin.Config.ResourceName, may not be empty string")
	}
	return nil
}

func (p *Plugin) register() error {
	conn, err := grpc.Dial(p.Config.KubeletSocket, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	defer conn.Close()
	if err != nil {
		return fmt.Errorf("device-plugin: cannot connect to kubelet service: %v", err)
	}
	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     p.Config.ServerSock,
		ResourceName: p.Config.ResourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return fmt.Errorf("device-plugin: cannot register to kubelet service: %v", err)
	}
	return nil
}

func (p *Plugin) startDeviceServer() error {
	if p.Deps.DevicePluginServer == nil {
		return fmt.Errorf("failed to provide Plugin.Deps.DevicePluginServer")
	}

	fi, err := os.Stat(p.Config.ListenEndpoint())
	if err == nil && (fi.Mode()&os.ModeSocket) != 0 {
		if err := os.Remove(p.Config.ListenEndpoint()); err != nil {
			return err
		}
	}

	sock, err := net.Listen("unix", p.Config.ListenEndpoint())
	if err != nil {
		return err
	}
	p.grpcServer = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(p.grpcServer, p.Deps.DevicePluginServer)

	p.Log.Infof("Starting Device Plugin's gRPC server listening on socket: %s", p.Config.ServerSock)
	go func() {
		if err := p.grpcServer.Serve(sock); err != nil {
			p.Log.Error("failed to start device plugin grpc server", p.Config.ListenEndpoint(), err)
		}
	}()
	// Check if the socket of device plugin server is operation
	conn, err := tools.SocketOperationCheck(p.Config.ListenEndpoint())
	if err != nil {
		return err
	}
	conn.Close()

	return nil
}

// Close is called when the plugin is being stopped
func (p *Plugin) Close() error {
	return p.IdempotentClose(plugintools.LoggingCloseFunc(p.Log, p, p.close))
}

func (p *Plugin) close() error {
	p.Log.Info("Close")
	p.grpcServer.GracefulStop()
	return nil
}
