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
	"sync"
	"time"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
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
	// POD UID is used as a unique key in clientConnections map
	// Second key is NetworkService name
	clientConnections map[string]map[string]*clientNetworkService
	sync.RWMutex
}

type nsmSocket struct {
	device      *pluginapi.Device
	socketPath  string
	stopChannel chan bool
	allocated   bool
}

// clientNetworkService struct represents requested by a NSM client NetworkService and its state, isInProgress true
// indicates that DataPlane programming operation is on going, so no duplicate request for Dataplane processing should occur.
type clientNetworkService struct {
	networkService       *netmesh.NetworkService
	ConnectionParameters *nsmconnect.ConnectionParameters
	// isInProgress indicates ongoing dataplane programming
	isInProgress bool
}

// RequestConnection accepts connection from NSM client and attempts to analyze requested info, call for Dataplane programming and
// return to NSM client result.
func (n *nsmClientEndpoints) RequestConnection(ctx context.Context, cr *nsmconnect.ConnectionRequest) (*nsmconnect.ConnectionAccept, error) {
	n.logger.Infof("received connection request id: %s from pod: %s/%s, requesting network service: %s for linux namespace: %s",
		cr.RequestId, cr.Metadata.Namespace, cr.Metadata.Name, cr.NetworkServiceName, cr.LinuxNamespace)

	// first check to see if requested NetworkService exists in objectStore
	ns := n.objectStore.GetNetworkService(cr.NetworkServiceName, cr.Metadata.Namespace)
	if ns == nil {
		// Unknown NetworkService fail Connection request
		return &nsmconnect.ConnectionAccept{
			Accepted:       false,
			AdmissionError: fmt.Sprintf("requested Network Service %s/%s does not exist", cr.Metadata.Namespace, cr.NetworkServiceName),
		}, status.Error(codes.NotFound, "requested network service not found")
	}

	// second check to see if requested NetworkService exists in n.clientConnections which means it is not first
	// Connection request
	if _, ok := n.clientConnections[cr.RequestId][cr.NetworkServiceName]; ok {
		// Since it is duplicate request, need to check if it is already inProgress
		if isInProgress(n.clientConnections[cr.RequestId][cr.NetworkServiceName]) {
			// Looks like dataplane programming is taking long time, responding client to wait and retry
			// TODO (sbezverk) nsm client should watch for this type of error and not fail but trigger exponential retry mechanism.
			return &nsmconnect.ConnectionAccept{
				Accepted:       false,
				AdmissionError: fmt.Sprintf("dataplane for requested Network Service %s/%s is still being programmed, retry", cr.Metadata.Namespace, cr.NetworkServiceName),
			}, status.Error(codes.AlreadyExists, "dataplane for requested network service is being programmed, retry")
		}
		// Request is not inProgress which means potentially a success can be returned
		// TODO (sbezverk) discuss this logic in case some corner cases might break it.
		return &nsmconnect.ConnectionAccept{
			Accepted:             true,
			ConnectionParameters: &nsmconnect.ConnectionParameters{},
		}, nil
	}
	// It is a new Connection request for known NetworkService, need to check if requested interface
	// parameters have a match with ones of known NetworkService. If not, return error
	reqInterfacesSorted := sortReqInterfaces(cr.Interface)

	// TODO (sbezverk) needs to be refactored for more sofisticated matching algorithm
	channel, found := findInterface(ns, reqInterfacesSorted)
	if !found {
		return &nsmconnect.ConnectionAccept{
			Accepted:       false,
			AdmissionError: fmt.Sprintf("no advertised channels for Network Service %s, support required interface", cr.NetworkServiceName),
		}, status.Error(codes.NotFound, "required interface type not found")
	}
	// Add new Connection Request into n.clientConnection, set as inProgress and call DataPlane programming func
	// and wait for complition.
	clientNS := clientNetworkService{
		networkService: &netmesh.NetworkService{
			Metadata: ns.Metadata,
			Channel:  []*netmesh.NetworkServiceChannel{channel},
		},
		isInProgress:         true,
		ConnectionParameters: &nsmconnect.ConnectionParameters{},
	}
	n.Lock()
	n.clientConnections[cr.RequestId][cr.NetworkServiceName] = &clientNS
	n.Unlock()

	// At this point we have all information to call Connection Request to NSE providing requested NetworkSerice
	// Once it is done and sucessfull call to dataplane programming function.
	// but it is for next PR

	// Simulating sucessfull end
	n.Lock()
	n.clientConnections[cr.RequestId][cr.NetworkServiceName].isInProgress = false
	n.Unlock()

	return &nsmconnect.ConnectionAccept{
		Accepted:             true,
		ConnectionParameters: n.clientConnections[cr.RequestId][cr.NetworkServiceName].ConnectionParameters,
	}, nil
}

func findInterface(ns *netmesh.NetworkService, reqInterfacesSorted []*common.Interface) (*netmesh.NetworkServiceChannel, bool) {
	for _, c := range ns.Channel {
		for _, i := range c.Interface {
			for _, iReq := range reqInterfacesSorted {
				if i.Type == iReq.Type {
					return c, true
				}
			}
		}
	}
	return nil, false
}

func sortReqInterfaces(reqInterfaces []*common.Interface) []*common.Interface {
	sortedInterfaces := make([]*common.Interface, len(reqInterfaces))
	for _, i := range reqInterfaces {
		sortedInterfaces[int(i.Preference)] = i
	}
	return sortedInterfaces
}

// TODO (sbezverk) Current assumption is that NSM client is requesting connection for  NetworkService
// from the same namespace. If it changes, refactor maybe required.
func isInProgress(networkService *clientNetworkService) bool {
	return networkService.isInProgress
}

func (n *nsmClientEndpoints) RequestDiscovery(ctx context.Context, cr *nsmconnect.DiscoveryRequest) (*nsmconnect.DiscoveryResponse, error) {
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
