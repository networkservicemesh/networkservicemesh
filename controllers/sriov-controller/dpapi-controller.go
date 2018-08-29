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
	"path"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	serverBasePath          = pluginapi.DevicePluginPath
	updateChannelBufferSize = 10
)

type registrationState int

const (
	notRegistered registrationState = iota
	registrationInProgress
	registered
)

type serviceInstanceController struct {
	serviceInstance
	regState registrationState
	sync.RWMutex
	socket             string
	networkServiceName string
	// regUpdateCh is used to signal changes in vfs map
	regUpdateCh chan struct{}
	// regStopCh is used to signal to stop advertise devices
	regStopCh chan struct{}
	// regDoneCh is used to confirm successful shutdown of WatchAndListen
	regDoneCh chan struct{}
	server    *grpc.Server
}

func newServiceInstanceController() *serviceInstanceController {
	si := serviceInstance{
		vfs: map[string]*VF{},
	}
	return &serviceInstanceController{si, notRegistered, sync.RWMutex{}, "", "",
		make(chan struct{},
			updateChannelBufferSize),
		make(chan struct{}),
		make(chan struct{}),
		nil}
}

// Run starts Network Service instance and wait for configuration messages
func (s *serviceInstanceController) Run() {
	logrus.Info("Started service instance controller, waiting for confiugration to register with the kubelet..")
	for {
		select {
		case <-s.stopCh:
			// shutdown received exiting wait loop
			logrus.Infof("Received shutdown message, network service %s is shutting down.", s.networkServiceName)
			s.regStopCh <- struct{}{}
			// Waiting for WatchAndList to complete
			<-s.regDoneCh
			// At this point all cleanup is done so can inform upstream
			close(s.doneCh)
			return
		case msg := <-s.configCh:
			switch msg.op {
			case operationAdd:
				s.processAddVF(msg)
			case operationDeleteEntry:
				s.processDeleteVF(msg)
			default:
				logrus.Errorf("error, recevied message with unknown operation %d", msg.op)
			}
		}
	}
}

// processAddVF checks if Network Service instance has already been registered, if not registration process gets triggered
// otherwise new VF gets added to the map and Update message send to refresh list of available VFs
func (s *serviceInstanceController) processAddVF(msg configMessage) {
	logrus.Infof("Network Service instance: %s, adding new VF, PCI address: %s", msg.vf.NetworkService, msg.pciAddr)
	if s.regState == notRegistered {
		logrus.Infof("service instance controller for %s has not yet been registered with kubelet, initiating registration process", msg.vf.NetworkService)
		s.regState = registrationInProgress
		go s.startDevicePlugin(msg)
	}
	s.Lock()
	s.vfs[msg.pciAddr] = &msg.vf
	s.Unlock()
	// Sending ListAndWatch notification of an update
	s.regUpdateCh <- struct{}{}
}

// processDeleteVF delete from VFs map deleted entry, then check if VFs are still left in Network Service instance
// if none left, shut down Network Service instance, otherwise send Update message to refresh the list of available VFs
func (s *serviceInstanceController) processDeleteVF(msg configMessage) {
	logrus.Infof("Network Service instance: %s, delete VF, PCI address: %s", msg.vf.NetworkService, msg.pciAddr)
	s.Lock()
	delete(s.vfs, msg.pciAddr)
	s.Unlock()
	// Sending ListAndWatch notification of an update
	s.regUpdateCh <- struct{}{}
}

// TODO (sbezverk) need to make sure that NetworkService name is complaint with dpapi nameing convention.
func (s *serviceInstanceController) startDevicePlugin(msg configMessage) {

	// All info for registration with kubelet is ready, attempting to register
	s.networkServiceName = msg.vf.NetworkService
	s.socket = path.Join(serverBasePath, strings.Replace(s.networkServiceName, "/", "-", -1)+".sock")

	// starting gRPC server for kubelet's Allocate and ListAndWatch calls
	s.startServer()

	logrus.Infof("attempting to register network service: %s on socket: %s", s.networkServiceName, s.socket)
	for s.regState != registered {
		logrus.Infof("Service Instance controller for %s attempting to register with kubelet", msg.vf.NetworkService)
		if err := s.register(); err != nil {
			logrus.Errorf("attempt to register with kubelet failed with error: %+v re-attempting in 10 seconds", err)
			time.Sleep(10 * time.Second)
		} else {
			s.regState = registered
			logrus.Infof("service instance controller: %s has been registered with kubelet", msg.vf.NetworkService)
		}
	}
}

func (s *serviceInstanceController) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (s *serviceInstanceController) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (s *serviceInstanceController) startServer() error {
	err := s.cleanup()
	if err != nil {
		return err
	}

	sock, err := net.Listen("unix", s.socket)
	if err != nil {
		return err
	}

	s.server = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(s.server, s)

	go s.server.Serve(sock)

	// Wait for server to start by launching a blocking connexion
	conn, err := dial(s.socket, 5*time.Second)
	if err != nil {
		return err
	}
	conn.Close()

	return nil
}

func (s *serviceInstanceController) cleanup() error {
	if err := os.Remove(s.socket); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// register registers service instance controller for the given network service with Kubelet.
func (s *serviceInstanceController) register() error {
	conn, err := dial(pluginapi.KubeletSocket, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(s.socket),
		ResourceName: s.networkServiceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		return err
	}
	return nil
}

func (s *serviceInstanceController) buildDeviceList(health string) []*pluginapi.Device {
	deviceList := []*pluginapi.Device{}
	s.Lock()
	defer s.Unlock()
	for _, vf := range s.vfs {
		device := pluginapi.Device{}
		device.ID = vf.VFIODevice
		device.Health = health
		deviceList = append(deviceList, &device)
	}

	return deviceList
}

// ListAndWatch converts VFs into device and list them
func (s *serviceInstanceController) ListAndWatch(e *pluginapi.Empty, d pluginapi.DevicePlugin_ListAndWatchServer) error {
	logrus.Infof("network service %s received ListandWatch from kubelet", s.networkServiceName)
	d.Send(&pluginapi.ListAndWatchResponse{Devices: s.buildDeviceList(pluginapi.Healthy)})
	for {
		select {
		case <-s.regStopCh:
			logrus.Infof("ListAndWatch of Network Service %s received shut down signal.", s.networkServiceName)
			// Informing kubelet that VFs which belong to network service are not useable now
			d.Send(&pluginapi.ListAndWatchResponse{Devices: s.buildDeviceList(pluginapi.Unhealthy)})
			close(s.regDoneCh)
			return nil
		case <-s.regUpdateCh:
			// Received a notification of a change in VFs resending updated list to kubelet
			d.Send(&pluginapi.ListAndWatchResponse{Devices: s.buildDeviceList(pluginapi.Healthy)})
			logrus.Infof("ListAndWatch of Network Service %s received update signal.", s.networkServiceName)
		}
	}
}

// Allocate which return list of devices.
func (s *serviceInstanceController) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	logrus.Infof("network service %s received Allocate from kubelet", s.networkServiceName)
	responses := pluginapi.AllocateResponse{}
	for _, req := range reqs.ContainerRequests {
		response := pluginapi.ContainerAllocateResponse{
			Devices: []*pluginapi.DeviceSpec{},
		}
		for _, id := range req.DevicesIDs {
			deviceSpec := pluginapi.DeviceSpec{}
			logrus.Infof("Allocation request for device: %s", id)
			if !s.checkVF(id) {
				return nil, fmt.Errorf("invalid allocation request: unknown device: %s", id)
			}
			deviceSpec.HostPath = id
			deviceSpec.ContainerPath = id
			deviceSpec.Permissions = "rw"
			response.Devices = append(response.Devices, &deviceSpec)
		}
		// Since the parent vfio device is also required to be visible in a container, adding it to the device list
		// so kubelet could do necessary arrangements.
		deviceSpec := pluginapi.DeviceSpec{}
		deviceSpec.HostPath = "/dev/vfio/vfio"
		deviceSpec.ContainerPath = "/dev/vfio/vfio"
		deviceSpec.Permissions = "rw"
		response.Devices = append(response.Devices, &deviceSpec)
		//
		responses.ContainerResponses = append(responses.ContainerResponses, &response)
	}

	return &responses, nil
}

func (s *serviceInstanceController) checkVF(id string) bool {
	for _, vf := range s.vfs {
		if vf.VFIODevice == id {
			return true
		}
	}
	return false
}

// dial establishes the gRPC communication with the registered device plugin.
func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
	if err != nil {
		return nil, err
	}

	return c, nil
}
