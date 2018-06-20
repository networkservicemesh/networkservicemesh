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

// Package core manages the lifecycle of all plugins (start, graceful
// shutdown) and defines the core lifecycle SPI. The core lifecycle SPI
// must be implemented by each plugin.

package nsmdp

import (
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/ligato/cn-infra/logging"

	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	socketBaseDir   = "/var/lib/networkservicemesh/"
	resourceName    = "nsm.ligato.io/socket"
	serverSock      = "nsm.ligato.io.sock"
	initDeviceCount = 10
	socketMask      = 0077
	folderMask      = 0777
)

type nsmSocket struct {
	device      *pluginapi.Device
	socketPath  string
	stopChannel chan bool
	allocated   bool
}

type nsmClientEndpoints struct {
	nsmSockets map[string]nsmSocket
	logger     logging.PluginLogger
}

// Define functions needed to meet the Kubernetes DevicePlugin API
func (n *nsmClientEndpoints) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	n.logger.Infof("GetDevicePluginOptions was called.")
	return &pluginapi.DevicePluginOptions{}, nil
}

func startClientServer(client *nsmSocket, logger logging.PluginLogger) {
	listenEndpoint := client.socketPath
	fi, err := os.Stat(listenEndpoint)
	if err == nil && (fi.Mode()&os.ModeSocket) != 0 {
		os.Remove(listenEndpoint)
	}
	if err != nil && !os.IsNotExist(err) {
		logger.Errorf("failure stat of socket file %s with error: %+v", client.socketPath, err)
		client.allocated = false
		return
	}

	unix.Umask(socketMask)
	sock, err := net.Listen("unix", listenEndpoint)
	if err != nil {
		logger.Errorf("failure to listen on socket %s with error: %+v", client.socketPath, err)
		client.allocated = false
		return
	}
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)

	logger.Infof("Starting Client gRPC server listening on socket: %s", serverSock)
	go grpcServer.Serve(sock)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := dial(ctx, listenEndpoint)
	if err != nil {
		logger.Errorf("failure to communicate with the socket %s with error: %+v", client.socketPath, err)
		client.allocated = false
	}
	defer conn.Close()
	defer cancel()
	logger.Infof("Client Server socket: %s is operational", listenEndpoint)

	// Wait for shutdown
	select {
	case <-client.stopChannel:
		logger.Infof("Server for socket %s received shutdown request", client.socketPath)
	}
	client.allocated = false
	client.stopChannel <- true
}

func (n *nsmClientEndpoints) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	n.logger.Info(" Allocate was called.")
	responses := pluginapi.AllocateResponse{}
	for _, req := range reqs.ContainerRequests {
		var mounts []*pluginapi.Mount
		for _, id := range req.DevicesIDs {
			if _, ok := n.nsmSockets[id]; ok {
				if n.nsmSockets[id].allocated {
					// Socket has been previsously used, since we did not get notification from
					// kubelet when POD using this socket went down, gRPC client's server
					// needs to be stopped.
					n.nsmSockets[id].stopChannel <- true
					// Wait for confirmation
					<-n.nsmSockets[id].stopChannel
					close(n.nsmSockets[id].stopChannel)
				}
				mount := &pluginapi.Mount{
					ContainerPath: socketBaseDir,
					HostPath:      path.Join(socketBaseDir, fmt.Sprintf("nsm-%s", id)),
					ReadOnly:      false,
				}
				n.nsmSockets[id] = nsmSocket{
					device:      &pluginapi.Device{ID: id, Health: pluginapi.Healthy},
					socketPath:  path.Join(mount.HostPath, serverSock),
					stopChannel: make(chan bool),
					allocated:   true,
				}
				if err := os.MkdirAll(mount.HostPath, folderMask); err == nil {
					// Starting Client's gRPC server and managed to create its host path.
					client := n.nsmSockets[id]
					go startClientServer(&client, n.logger)
					mounts = append(mounts, mount)
				}
			}
		}
		response := pluginapi.ContainerAllocateResponse{
			Mounts: mounts,
		}
		responses.ContainerResponses = append(responses.ContainerResponses, &response)
	}
	return &responses, nil
}

func (n *nsmClientEndpoints) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	n.logger.Infof("ListAndWatch was called with s: %+v", s)
	for {
		resp := new(pluginapi.ListAndWatchResponse)
		for _, dev := range n.nsmSockets {
			resp.Devices = append(resp.Devices, dev.device)
		}
		if err := s.Send(resp); err != nil {
			n.logger.Errorf("Failed to send response to kubelet: %v\n", err)
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (n *nsmClientEndpoints) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	n.logger.Infof("PreStartContainer was called.")
	return &pluginapi.PreStartContainerResponse{}, nil
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
		Endpoint:     serverSock,
		ResourceName: resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return fmt.Errorf("device-plugin: cannot register to kubelet service: %v", err)
	}
	return nil
}

func dial(ctx context.Context, unixSocketPath string) (*grpc.ClientConn, error) {
	c, err := grpc.DialContext(ctx, unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}
	return c, nil
}

func startDeviceServer(nsm *nsmClientEndpoints) error {
	// Initial socket clean up
	listenEndpoint := path.Join(pluginapi.DevicePluginPath, serverSock)
	fi, err := os.Stat(listenEndpoint)
	if err == nil && (fi.Mode()&os.ModeSocket) != 0 {
		if err := os.Remove(listenEndpoint); err != nil {
			return err
		}
	}

	sock, err := net.Listen("unix", listenEndpoint)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(grpcServer, nsm)

	nsm.logger.Infof("Starting Device Plugin's gRPC server listening on socket: %s", serverSock)
	go grpcServer.Serve(sock)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := dial(ctx, listenEndpoint)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer cancel()
	nsm.logger.Infof("Socket: %s is operational", listenEndpoint)
	return nil
}

func newNSMClientEndpoints(logger logging.PluginLogger) *nsmClientEndpoints {
	nsm := &nsmClientEndpoints{
		nsmSockets: map[string]nsmSocket{},
		logger:     logger,
	}
	for i := 0; i < initDeviceCount; i++ {
		nsm.nsmSockets[strconv.Itoa(i)] = nsmSocket{device: &pluginapi.Device{ID: strconv.Itoa(i), Health: pluginapi.Healthy}}
	}
	return nsm
}

// NewNSMDevicePlugin registers and starts Kubelet's device plugin
func NewNSMDevicePlugin(logger logging.PluginLogger) error {
	nsm := newNSMClientEndpoints(logger)
	if err := startDeviceServer(nsm); err != nil {
		return err
	}
	// Registers with Kubelet.
	err := Register(pluginapi.KubeletSocket)

	return err
}
