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

package nsmserver

import (
	"fmt"
	"net"
	"os"
	"path"
	"time"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/ligato/networkservicemesh/plugins/logger"
	"github.com/ligato/networkservicemesh/plugins/objectstore"
	"golang.org/x/net/context"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type nsmClientEndpoints struct {
	nsmSockets  map[string]nsmSocket
	logger      logger.FieldLoggerPlugin
	objectStore objectstore.Interface
}

type nsmSocket struct {
	device      *pluginapi.Device
	socketPath  string
	stopChannel chan bool
	allocated   bool
}

func (n nsmClientEndpoints) RequestConnection(ctx context.Context, cr *nsmconnect.ConnectionRequest) (*nsmconnect.ConnectionAccept, error) {
	return nil, status.Error(codes.InvalidArgument, "Not Implemented...")
}

func (n nsmClientEndpoints) RequestDiscovery(ctx context.Context, cr *nsmconnect.DiscoveryRequest) (*nsmconnect.DiscoveryResponse, error) {
	n.logger.Info("received Discovery request")
	networkService := n.objectStore.ListNetworkServices()
	n.logger.Infof("preparing Discovery response, number of returning NetworkServices: %d", len(networkService))
	resp := &nsmconnect.DiscoveryResponse{
		NetworkService: networkService,
	}
	return resp, nil
}

func (n *nsmClientEndpoints) RequestAdvertiseChannel(ctx context.Context, cr *nsmconnect.ChannelAdvertiseRequest) (*nsmconnect.ChannelAdvertiseResponse, error) {
	n.logger.Printf("received Channel advertisement...")
	for _, c := range cr.NetmeshChannel {
		n.logger.Infof("For NetworkService: %s channel: %s channel's socket location: %s", c.NetworkServiceName, c.Metadata.Name, c.SocketLocation)
		networkServiceName := c.NetworkServiceName
		networkServiceNamespace := "default"
		if c.Metadata.Namespace != "" {
			networkServiceNamespace = c.Metadata.Namespace
		} else {
			c.Metadata.Namespace = "default"
		}

		networkService := n.objectStore.GetNetworkService(networkServiceName, networkServiceNamespace)
		if networkService != nil {
			n.logger.Infof("Found existing NetworkService %s/%s in the Object Store, will add channel %s to its list of channels",
				networkServiceName, networkServiceNamespace, c.Metadata.Name)
			// Since it was discovered that NetworkService Object exists, calling method to add the channel to NetworkService.
			if err := n.objectStore.AddChannelToNetworkService(networkServiceName, networkServiceNamespace, c); err != nil {
				n.logger.Error("failed to add channel %s/%s to network service %s with error: %+v", networkServiceNamespace, networkServiceName, c.Metadata.Name, err)
				return &nsmconnect.ChannelAdvertiseResponse{Success: false}, err
			}
			n.logger.Infof("Channel %s/%s has been successfully added to network service %s/%s in the Object Store",
				c.Metadata.Namespace, c.Metadata.Name, networkServiceName, networkServiceNamespace)
		} else {
			n.logger.Infof("NetworkService %s/%s is not found in the Object Store", networkServiceNamespace, networkServiceName)
			return &nsmconnect.ChannelAdvertiseResponse{Success: false}, fmt.Errorf("NetworkService %s/%s is not found in the Object Store",
				networkServiceNamespace, networkServiceName)
		}
	}
	return &nsmconnect.ChannelAdvertiseResponse{Success: true}, nil
}

// Define functions needed to meet the Kubernetes DevicePlugin API
func (n *nsmClientEndpoints) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	n.logger.Infof("GetDevicePluginOptions was called.")
	return &pluginapi.DevicePluginOptions{}, nil
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
					ContainerPath: SocketBaseDir,
					HostPath:      path.Join(SocketBaseDir, fmt.Sprintf("nsm-%s", id)),
					ReadOnly:      false,
				}
				n.nsmSockets[id] = nsmSocket{
					device:      &pluginapi.Device{ID: id, Health: pluginapi.Healthy},
					socketPath:  path.Join(mount.HostPath, ServerSock),
					stopChannel: make(chan bool),
					allocated:   true,
				}
				if err := os.MkdirAll(mount.HostPath, folderMask); err == nil {
					// Starting Client's gRPC server and managed to create its host path.
					go startClientServer(id, n)
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

func startClientServer(id string, endpoints *nsmClientEndpoints) {
	client := endpoints.nsmSockets[id]
	logger := endpoints.logger
	listenEndpoint := client.socketPath
	// TODO (sbezverk) make it as a function
	fi, err := os.Stat(listenEndpoint)
	if err == nil && (fi.Mode()&os.ModeSocket) != 0 {
		if err := os.Remove(listenEndpoint); err != nil {
			logger.Error("Cannot remove listen endpoint", listenEndpoint, err)
		}
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
	// PLugging NSM client Connection methods
	nsmconnect.RegisterClientConnectionServer(grpcServer, endpoints)
	logger.Infof("Starting Client gRPC server listening on socket: %s", ServerSock)
	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			logger.Fatalln("unable to start client grpc server: ", ServerSock, err)
		}
	}()

	if err := socketOperationCheck(listenEndpoint); err != nil {
		logger.Errorf("failure to communicate with the socket %s with error: %+v", client.socketPath, err)
		client.allocated = false
		return
	}
	logger.Infof("Client Server socket: %s is operational", listenEndpoint)

	// Wait for shutdown
	select {
	case <-client.stopChannel:
		logger.Infof("Server for socket %s received shutdown request", client.socketPath)
	}
	client.allocated = false
	client.stopChannel <- true
}

func socketOperationCheck(listenEndpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := dial(ctx, listenEndpoint)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer cancel()

	return nil
}
