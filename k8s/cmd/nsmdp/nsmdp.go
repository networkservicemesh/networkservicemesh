// Copyright 2018-2019 Red Hat, Inc.
// Copyright (c) 2018-2019 Cisco and/or its affiliates.
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
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/spanhelper"
	"net"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/nsmdapi"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/nsmd"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/serviceregistry"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

const (
	// SocketBaseDir defines the location of NSM client socket
	resourceName = "networkservicemesh.io/socket"
	// ServerSock defines the name of NSM client socket
	ServerSock = "networkservicemesh.io.sock"

	// A number of devices we have in buffer for use, so we hold extra DeviceBuffer count of deviceids send to kubelet.
	DeviceBuffer = 30

	// Send device ids to kubelet every N seconds.
	KubeletNotifyDelay = 30 * time.Second
)

type nsmClientEndpoints struct {
	serviceRegistry serviceregistry.ServiceRegistry
	resp            *pluginapi.ListAndWatchResponse
	devs            map[string]*pluginapi.Device
	pluginApi       *pluginapi.DevicePlugin_ListAndWatchServer
	mutext          sync.Mutex
	clientId        int
	insecure        bool
	span            spanhelper.SpanHelper
}

func (n *nsmClientEndpoints) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	span := spanhelper.SpanHelperFromContextCopySpan(ctx, n.span, "Allocate")
	defer span.Finish()

	span.Logger().Infof("Client request for nsmdp resource... %v", proto.MarshalTextString(reqs))
	responses := &pluginapi.AllocateResponse{}

	var mounts []*pluginapi.Mount
	if !n.insecure {
		mounts = append(mounts, &pluginapi.Mount{
			ContainerPath: "/run/spire/sockets",
			HostPath:      "/run/spire/sockets",
			ReadOnly:      true,
		})
	}

	for _, req := range reqs.ContainerRequests {
		id := req.DevicesIDs[0]
		span.Logger().Infof("Requesting Workspace, device ID: %s", id)
		workspace, err := nsmd.RequestWorkspace(span.Context(), n.serviceRegistry, id)
		span.Logger().Infof("Received Workspace %v", workspace)
		if err != nil {
			span.Logger().Errorf("error talking to nsmd: %v", err)
		} else {
			mount := &pluginapi.Mount{
				ContainerPath: workspace.ClientBaseDir,
				HostPath:      workspace.HostBasedir + workspace.Workspace,
				ReadOnly:      false,
			}
			envs := map[string]string{
				nsmd.NsmDevicePluginEnv:   "true",
				common.NsmServerSocketEnv: mount.ContainerPath + workspace.NsmServerSocket,
				common.NsmClientSocketEnv: mount.ContainerPath + workspace.NsmClientSocket,
				common.WorkspaceEnv:       workspace.ClientBaseDir,
			}

			if n.insecure {
				envs[tools.InsecureEnv] = strconv.FormatBool(true)
			}

			responses.ContainerResponses = append(responses.ContainerResponses, &pluginapi.ContainerAllocateResponse{
				Mounts: append(mounts, mount),
				Envs:   envs,
			})
			span.LogObject("responses", responses)
		}
	}
	span.Logger().Infof("AllocateResponse: %v", responses)
	return responses, nil
}

// Register registers
func Register(ctx context.Context, kubeletEndpoint string) error {
	conn, err := tools.DialUnixInsecure(ctx, kubeletEndpoint)
	if err != nil {
		return fmt.Errorf("device-plugin: cannot connect to kubelet service: %v", err)
	}
	defer func() { _ = conn.Close() }()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     ServerSock,
		ResourceName: resourceName,
	}

	_, err = client.Register(ctx, reqt)
	if err != nil {
		return fmt.Errorf("device-plugin: cannot register to kubelet service: %v", err)
	}
	return nil
}

// Define functions needed to meet the Kubernetes DevicePlugin API
func (n *nsmClientEndpoints) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (n *nsmClientEndpoints) PreStartContainer(ctx context.Context, info *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	logrus.Infof("Pre start container called... %v ", info)
	return &pluginapi.PreStartContainerResponse{}, nil
}

func enumWorkspaces(ctx context.Context, serviceRegistry serviceregistry.ServiceRegistry) (*nsmdapi.EnumConnectionReply, error) {
	client, con, err := serviceRegistry.NSMDApiClient()
	if err != nil {
		logrus.Fatalf("Failed to connect to NSMD: %+v...", err)
	}
	defer con.Close()
	reply, err := client.EnumConnection(ctx, &nsmdapi.EnumConnectionRequest{})
	if err != nil {
		return nil, err
	}
	return reply, nil
}

func indexOf(slice []string, value string) int {
	for i, v := range slice {
		if v == value {
			return i
		}
	}
	return -1
}

func (n *nsmClientEndpoints) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	logrus.Infof("ListAndWatch was called with s: %+v", s)
	n.pluginApi = &s

	// Restore state from NSMD
	for {
		n.receiveWorkspaces()
		n.sendDeviceUpdate()

		// Sleep before next notification.
		time.Sleep(KubeletNotifyDelay)
	}
}

func startDeviceServer(ctx context.Context, nsm *nsmClientEndpoints) error {
	span := spanhelper.SpanHelperFromContext(ctx, "start-device-server")
	defer span.Finish()
	listenEndpoint := path.Join(pluginapi.DevicePluginPath, ServerSock)
	span.LogObject("listen-endpoint", listenEndpoint)
	if err := tools.SocketCleanup(listenEndpoint); err != nil {
		return err
	}
	sock, err := net.Listen("unix", listenEndpoint)
	if err != nil {
		return err
	}

	grpcServer := tools.NewServerInsecure()

	pluginapi.RegisterDevicePluginServer(grpcServer, nsm)

	span.Logger().Infof("Starting Device Plugin's gRPC server listening on socket: %s", ServerSock)
	go func() {
		if err := grpcServer.Serve(sock); err != nil {
			span.Logger().Error("failed to start device plugin grpc server", listenEndpoint, err)
		}
	}()
	// Check if the socket of device plugin server is operation
	conn, err := tools.DialUnixInsecure(span.Context(), listenEndpoint)
	if err != nil {
		return err
	}
	_ = conn.Close()

	return nil
}

func waitForNsmdAvailable(ctx context.Context) {
	for {
		if tools.WaitForPortAvailable(ctx, "unix", nsmd.ServerSock, 100*time.Millisecond) == nil {
			break
		}
	}
}

// NewNSMDeviceServer registers and starts Kubelet's device plugin
func NewNSMDeviceServer(ctx context.Context, serviceRegistry serviceregistry.ServiceRegistry) error {
	span := spanhelper.SpanHelperFromContext(ctx, "start.device.server")
	defer span.Finish()
	waitForNsmdAvailable(span.Context())

	insecure, err := tools.IsInsecure()
	if err != nil {
		return err
	}

	nsm := &nsmClientEndpoints{
		serviceRegistry: serviceRegistry,
		resp:            new(pluginapi.ListAndWatchResponse),
		devs:            map[string]*pluginapi.Device{},
		insecure:        insecure,
		span:            span,
	}

	for i := 0; i < DeviceBuffer; i++ {
		nsm.addClientDevice(span)
	}

	if err := startDeviceServer(span.Context(), nsm); err != nil {
		return err
	}
	// Registers with Kubelet.
	return Register(span.Context(), pluginapi.KubeletSocket)
}

func (nsm *nsmClientEndpoints) addClientDevice(span spanhelper.SpanHelper) {
	nsm.mutext.Lock()
	defer nsm.mutext.Unlock()

	for {
		nsm.clientId++
		span.LogObject("clientId", nsm.clientId)

		id := fmt.Sprintf("nsm-%d", nsm.clientId)
		if _, ok := nsm.devs[id]; ok {
			// Item already exists, increment to new one.
			continue
		}
		// Add a new device Id
		dev := &pluginapi.Device{
			ID:     id,
			Health: pluginapi.Healthy,
		}
		span.LogObject("device", dev)
		nsm.devs[id] = dev
		nsm.resp.Devices = append(nsm.resp.Devices, dev)
		break
	}

}

func (n *nsmClientEndpoints) sendDeviceUpdate() {
	n.mutext.Lock()
	defer n.mutext.Unlock()
	if n.pluginApi != nil {
		api := *n.pluginApi
		n.span.Logger().Infof("Send device plugins intfo %v", n.resp)
		if err := api.Send(n.resp); err != nil {
			n.span.Logger().Errorf("Failed to send response to kubelet: %v\n", err)
		}
	}
}

func (n *nsmClientEndpoints) receiveWorkspaces() {
	for {
		reply, err := enumWorkspaces(n.span.Context(), n.serviceRegistry)
		if err != nil {
			n.span.Logger().Errorf("Error receive devices from NSM. %v", err)
			// Make a fast delay to faster startup of NSMD.
			<-time.After(100 * time.Millisecond)
			continue
		}
		n.mutext.Lock()

		// Check we had all workspaces in our update list
		// This list could be changed in case both NSMDp and NSMD restart.
		for _, w := range reply.GetWorkspace() {
			if len(w) > 0 {
				if _, ok := n.devs[w]; !ok {
					// We do not have this one in list of our devices
					dev := &pluginapi.Device{
						ID:     w,
						Health: pluginapi.Healthy,
					}
					n.devs[w] = dev
					n.resp.Devices = append(n.resp.Devices, dev)
				}
			}
		}
		n.mutext.Unlock()
		// Be sure we have enought deviced ids allocated.
		for len(n.resp.Devices) < len(reply.GetWorkspace())+DeviceBuffer {
			n.addClientDevice(n.span)
		}

		break
	}
}
