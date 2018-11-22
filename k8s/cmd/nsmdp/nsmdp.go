// Copyright 2018 Red Hat, Inc.
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

package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/ligato/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	// SocketBaseDir defines the location of NSM client socket
	resourceName = "nsm.ligato.io/socket"
	// ServerSock defines the name of NSM client socket
	ServerSock = "nsm.ligato.io.sock"
)

type nsmClientEndpoints struct {
}

func (n *nsmClientEndpoints) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	logrus.Info("Client request for nsmdp resource...")
	responses := &pluginapi.AllocateResponse{}
	for range reqs.ContainerRequests {
		workspace, err := nsmd.RequestWorkspace()
		logrus.Infof("Received Workspace %v", workspace)
		if err != nil {
			logrus.Errorf("error talking to nsmd: %v", err)
		} else {
			mount := &pluginapi.Mount{
				ContainerPath: workspace.ClientBaseDir,
				HostPath:      workspace.HostBasedir + workspace.Workspace,
				ReadOnly:      false,
			}
			responses.ContainerResponses = append(responses.ContainerResponses, &pluginapi.ContainerAllocateResponse{
				Mounts: []*pluginapi.Mount{mount},
				Envs: map[string]string{
					nsmd.NsmDevicePluginEnv: "true",
					nsmd.NsmServerSocketEnv: mount.ContainerPath + workspace.NsmServerSocket,
					nsmd.NsmClientSocketEnv: mount.ContainerPath + workspace.NsmClientSocket,
				},
			})
		}
	}
	logrus.Infof("AllocateResponse: %v", responses)
	return responses, nil
}

// Register registers
func Register(kubeletEndpoint string) error {
	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithInsecure(),
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
		Endpoint:     ServerSock,
		ResourceName: resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return fmt.Errorf("device-plugin: cannot register to kubelet service: %v", err)
	}
	return nil
}

// Define functions needed to meet the Kubernetes DevicePlugin API
func (n *nsmClientEndpoints) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (n *nsmClientEndpoints) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (n *nsmClientEndpoints) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	logrus.Infof("ListAndWatch was called with s: %+v", s)
	for {
		resp := new(pluginapi.ListAndWatchResponse)
		for dev := 1; dev < 10; dev++ {
			resp.Devices = append(resp.Devices, &pluginapi.Device{
				ID:     fmt.Sprintf("%d", dev),
				Health: pluginapi.Healthy,
			})
		}
		if err := s.Send(resp); err != nil {
			logrus.Errorf("Failed to send response to kubelet: %v\n", err)
		}
		time.Sleep(30 * time.Second)
	}
}

func startDeviceServer(nsm *nsmClientEndpoints) error {
	listenEndpoint := path.Join(pluginapi.DevicePluginPath, ServerSock)
	if err := tools.SocketCleanup(listenEndpoint); err != nil {
		return err
	}
	sock, err := net.Listen("unix", listenEndpoint)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(grpcServer, nsm)

	logrus.Infof("Starting Device Plugin's gRPC server listening on socket: %s", ServerSock)
	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logrus.Error("failed to start device plugin grpc server", listenEndpoint, err)
		}
	}()
	// Check if the socket of device plugin server is operation
	conn, err := tools.SocketOperationCheck(listenEndpoint)
	if err != nil {
		return err
	}
	conn.Close()

	return nil
}

func waitForNsmdAvailable() {
	for {
		if tools.WaitForPortAvailable(context.Background(), "unix", nsmd.ServerSock, 100*time.Millisecond) == nil {
			break
		}
	}
}

// NewNSMDeviceServer registers and starts Kubelet's device plugin
func NewNSMDeviceServer() error {
	waitForNsmdAvailable()
	nsm := &nsmClientEndpoints{}
	if err := startDeviceServer(nsm); err != nil {
		return err
	}
	// Registers with Kubelet.
	err := Register(pluginapi.KubeletSocket)

	return err
}

func main() {
	err := NewNSMDeviceServer()

	if err != nil {
		logrus.Errorf("failed to start server: %v", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		wg.Done()
	}()
	wg.Wait()
}
