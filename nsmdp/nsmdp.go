package nsmdp

import (
	"context"
	"crypto/rand"
	"fmt"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	"networkservicemesh/deviceplugin"
	"os"
)

const (
	SocketBaseDir = "/var/lib/networkservicemesh/"
)

type NSMDevicePlugin struct {
	*deviceplugin.DevicePlugin
	devs       map[string]*NSMDevice
	updatedevs chan *NSMDevice
	stop       chan interface{}
}

type NSMDevice struct {
	*pluginapi.Device
	token string
}

const (
	resourceName    = "nsm.ligato.io"
	serverSock      = pluginapi.DevicePluginPath + "nsm.ligato.io.sock"
	initDeviceCount = 10
)

func NewNSMDevicePlugin() *NSMDevicePlugin {
	n := &NSMDevicePlugin{
		DevicePlugin: deviceplugin.NewDevicePlugin(serverSock, resourceName),
		devs:         make(map[string]*NSMDevice),
		stop:         make(chan interface{}),
		updatedevs:   make(chan *NSMDevice, 10),
	}
	for i := uint(0); i < initDeviceCount; i++ {
		dev := &NSMDevice{
			Device: &pluginapi.Device{
				ID:     "NSM_" + string(i),
				Health: pluginapi.Healthy,
			},
			token: generateToken(),
		}
		n.updatedevs <- dev
	}
	return n
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (n *NSMDevicePlugin) Stop() error {
	err := n.DevicePlugin.Stop()
	if err != nil {
		return err
	}
	close(n.stop)
	return nil

}

func (n *NSMDevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	responses := pluginapi.AllocateResponse{}
	for _, req := range reqs.ContainerRequests {
		var mounts []*pluginapi.Mount
		for _, id := range req.DevicesIDs {
			_, ok := n.devs[id]
			if !ok {
				return nil, fmt.Errorf("invalid allocation request: unknown device: %s", id)
			}
			mount := &pluginapi.Mount{
				ContainerPath: SocketBaseDir + id,
				HostPath:      SocketBaseDir + id,
			}
			os.MkdirAll(mount.HostPath, 0777)
			// FIXME - Create socket files for Pod to NSM communication
			mounts = append(mounts, mount)
		}
		response := pluginapi.ContainerAllocateResponse{
			Mounts: mounts,
		}
		responses.ContainerResponses = append(responses.ContainerResponses, &response)
	}
	return &responses, nil
}

func (n *NSMDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	for {
		select {
		case <-n.stop:
			return nil
		case dev := <-n.updatedevs:
			n.devs[dev.ID] = dev
			devs := []*pluginapi.Device{dev.Device}
			s.Send(&pluginapi.ListAndWatchResponse{Devices: devs})
		}
	}
}
