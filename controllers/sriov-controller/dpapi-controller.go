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
	"encoding/json"
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

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

const (
	serverBasePath          = pluginapi.DevicePluginPath
	updateChannelBufferSize = 10
	containerConfigFilePath = "/var/lib/networkservicemesh"
	nsmVFsPrefix            = "NSM_VFS_"
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

func newServiceInstanceController(configCh chan configMessage, stopCh, doneCh chan struct{}) *serviceInstanceController {
	si := serviceInstance{
		vfs:      map[string]*VF{},
		configCh: configCh,
	}
	sic := &serviceInstanceController{
		serviceInstance: si,
		regState:        notRegistered,
		regUpdateCh:     make(chan struct{}, updateChannelBufferSize),
		regStopCh:       stopCh,
		regDoneCh:       doneCh,
	}
	sic.RWMutex = sync.RWMutex{}

	return sic
}

// Run starts Network Service instance and wait for configuration messages
func (s *serviceInstanceController) Run() {
	logrus.Info("Started service instance controller, waiting for configuration to register with the kubelet..")
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
				logrus.Errorf("error, received message with unknown operation %d", msg.op)
			}
		}
	}
}

// processAddVF checks if Network Service instance has already been registered, if not registration process gets triggered
// otherwise new VF gets added to the map and Update message send to refresh list of available VFs
func (s *serviceInstanceController) processAddVF(msg configMessage) {
	logrus.Infof("Network Service instance: %s, adding new VF, PCI address: %s", msg.vf.NetworkService, msg.pciAddr)
	if s.regState == notRegistered {
		s.Lock()
		logrus.Infof("service instance controller for %s has not yet been registered with kubelet, initiating registration process", msg.vf.NetworkService)
		s.regState = registrationInProgress
		s.Unlock()
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
	defer s.Unlock()
	delete(s.vfs, msg.pciAddr)
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

	s.server = tools.NewServer(context.Background())
	pluginapi.RegisterDevicePluginServer(s.server, s)

	go s.server.Serve(sock)

	// Wait for server to start by launching a blocking connection
	conn, err := tools.DialUnix(context.Background(), s.socket)
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
	conn, err := tools.DialUnix(context.Background(), pluginapi.KubeletSocket)
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
	deviceList := make([]*pluginapi.Device, 0)
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
			d.Send(&pluginapi.ListAndWatchResponse{
				Devices: []*pluginapi.Device{}})
			close(s.regDoneCh)
			return nil
		case <-s.regUpdateCh:
			// Received a notification of a change in VFs resending updated list to kubelet
			d.Send(&pluginapi.ListAndWatchResponse{Devices: s.buildDeviceList(pluginapi.Healthy)})
			logrus.Infof("ListAndWatch of Network Service %s received update signal.", s.networkServiceName)
		}
	}
}

// vfioConfig is stuct used to store vfio device specific information
type vfioConfig struct {
	VFIODevice string `yaml:"vfioDevice" json:"vfioDevice"`
	PCIAddr    string `yaml:"pciAddr" json:"pciAddr"`
}

// getVFIODevSpecs looks for vfio device's specs, example PCI address and return vfioConfig instance.
// In future other parameters could be added, example NUMATopology, CPUPinning, etc.
func (s *serviceInstanceController) getVFIODevSpecs(vfioDev string) (vfioConfig, error) {
	for _, vfio := range s.serviceInstance.vfs {
		if vfio.VFIODevice == vfioDev {
			return vfioConfig{
				VFIODevice: vfioDev,
				PCIAddr:    vfio.PCIAddr,
			}, nil
		}
	}
	return vfioConfig{}, fmt.Errorf("vfio %s device is not found", vfioDev)
}

// Allocate which return list of devices.
func (s *serviceInstanceController) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	logrus.Infof("network service %s received Allocate from kubelet", s.networkServiceName)
	// Allocating Device Plugin API response struct
	responses := pluginapi.AllocateResponse{}
	// Allocating slice of vfioConfigs, the content of this slice will be saved as a json file and make available to POD requesting
	// the network service.
	vfioDevs := make([]vfioConfig, 0)
	// Bulding per network service key for Env variable, it will point to the network service configuration
	// file.
	networkServiceName := strings.ToLower(strings.Split(s.networkServiceName, "/")[1])
	networkServiceName = strings.Replace(networkServiceName, "-", "_", -1)
	if strings.HasPrefix(networkServiceName, "sriov_") {
		networkServiceName = strings.Split(networkServiceName, "sriov_")[1]
	}
	key := nsmVFsPrefix + networkServiceName
	// Creating config file which will be passed to the POD. The config file name is composed
	// from Network Service name + vfio group ID taken from the first device in Allocate Request.
	// Example: If Network Service is vlan10 and AllocateRequest has /dev/vfio/67, then the config
	// file name will be vlan10_67.json.
	// If a file with the same name already exists, os.Create will truncate it and previous content will be lost.
	_, groupID := path.Split(reqs.ContainerRequests[0].DevicesIDs[0])
	configFileName := fmt.Sprintf("%s_%s.json", networkServiceName, groupID)
	configFile, err := os.Create(path.Join(containersConfigPath, configFileName))
	if err != nil {
		return nil, fmt.Errorf("fail to create network services config file with error: %+v", err)
	}
	defer configFile.Close()
	for _, req := range reqs.ContainerRequests {
		response := pluginapi.ContainerAllocateResponse{
			Devices: []*pluginapi.DeviceSpec{},
			// Adding env variable for requested network service, the key is composed as "NSM_VFS_"+ {network service name}
			// excluding organization prefix.
			// Environment variable points to the location of the network service specific configuration file
			Envs: map[string]string{
				key: path.Join(containerConfigFilePath, configFileName),
			},
			Mounts: []*pluginapi.Mount{
				{
					// Adding this specific network service configuration file into the container
					ContainerPath: path.Join(containerConfigFilePath, configFileName),
					HostPath:      configFile.Name(),
					ReadOnly:      true,
				},
			},
		}
		for _, id := range req.DevicesIDs {
			deviceSpec := pluginapi.DeviceSpec{}
			logrus.Infof("Allocation request for device: %s", id)
			if !s.checkVF(id) {
				return nil, fmt.Errorf("allocation request failure, unknown device: %s", id)
			}
			deviceSpec.HostPath = id
			deviceSpec.ContainerPath = id
			deviceSpec.Permissions = "rw"
			response.Devices = append(response.Devices, &deviceSpec)
			// Getting vfio device specific specifications and storing it in the slice. The slice
			// will be marshaled into json and passed to requesting POD as a mount.
			vfioDev, err := s.getVFIODevSpecs(id)
			if err != nil {
				return nil, fmt.Errorf("allocation request failure, unable to get device %s specs with error: %+v", id, err)
			}
			vfioDevs = append(vfioDevs, vfioDev)
		}
		// Since the parent vfio device is also required to be visible in a container, adding it to the device list
		// so kubelet could do necessary arrangements.
		deviceSpec := pluginapi.DeviceSpec{}
		deviceSpec.HostPath = "/dev/vfio/vfio"
		deviceSpec.ContainerPath = "/dev/vfio/vfio"
		deviceSpec.Permissions = "rw"
		response.Devices = append(response.Devices, &deviceSpec)

		responses.ContainerResponses = append(responses.ContainerResponses, &response)
	}
	// Last step is to store vfio devices specification in the  network service configuration file.
	configBytes, err := json.Marshal(&vfioDevs)
	if err != nil {
		return nil, fmt.Errorf("allocation request failure, unable to marshal config file with error: %+v", err)
	}
	if _, err := configFile.Write(configBytes); err != nil {
		return nil, fmt.Errorf("allocation request failure, unable to save config file with error: %+v", err)
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
