package deviceplugin

import (
	"context"
	"net"
	"net/url"
	"strconv"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/crossapi/chains/nsmgr"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/crossapi/chains/nsmgr/peer_tracker"
	"github.com/networkservicemesh/networkservicemesh/new/sdk/tools/serialize"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type NsmDevicePluginServer interface {
	networkservice.NetworkServiceServer
	pluginapi.DevicePluginServer
}

type nsmgrDevicePlugin struct {
	nsmgr.Nsmgr
	devices               map[string]*pluginapi.Device
	allocatedDevices      map[string]*pluginapi.Device
	executor              serialize.Executor
	insecure              bool
	reallocate            func(u *url.URL)
	listAndWatchListeners []pluginapi.DevicePlugin_ListAndWatchServer
}

func NewServer(name string, insecure bool, registryCC *grpc.ClientConn) NsmDevicePluginServer {
	rv := &nsmgrDevicePlugin{
		devices:          make(map[string]*pluginapi.Device, DeviceBuffer),
		allocatedDevices: make(map[string]*pluginapi.Device, DeviceBuffer),
		executor:         serialize.NewExecutor(),
		insecure:         insecure,
	}
	// TODO - Fix applying peer_tracker here
	// rv.Nsmgr = peer_tracker.NewServer(nsmgr.NewEndpoint(name, registryCC), &rv.reallocate)
	rv.Nsmgr = peer_tracker.NewServer(nsmgr.NewNsmgr(name, registryCC), &rv.reallocate)
	rv.resizeDevicePool()
	return rv
}

func (n *nsmgrDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	n.executor.Exec(func() {
		n.listAndWatchListeners = append(n.listAndWatchListeners, s)
		listAndWatchResponse := &pluginapi.ListAndWatchResponse{}
		for _, device := range n.devices {
			listAndWatchResponse.Devices = append(listAndWatchResponse.Devices, device)
		}
		for _, listAndWatchListener := range n.listAndWatchListeners {
			listAndWatchListener.Send(listAndWatchResponse)
		}
	})

	<-s.Context().Done()
	n.executor.Exec(func() {
		var listAndWatchListeners []pluginapi.DevicePlugin_ListAndWatchServer
		for _, listAndWatchListener := range n.listAndWatchListeners {
			if listAndWatchListener != s {
				listAndWatchListeners = append(listAndWatchListeners, listAndWatchListener)
			}
		}
	})
	return nil
}

func (n *nsmgrDevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	containerResponses := []*pluginapi.ContainerAllocateResponse{}
	for _, req := range reqs.GetContainerRequests() {
		for _, deviceid := range req.GetDevicesIDs() {
			// Close any existing connection from previous allocation
			n.reallocate(&url.URL{
				Scheme: "unix",
				Path:   localServerSocketFile(deviceid),
			})
			mounts := []*pluginapi.Mount{
				{
					ContainerPath: containerDeviceDirectory(deviceid),
					HostPath:      hostDeviceDirectory(deviceid),
					ReadOnly:      false,
				},
			}
			envs := map[string]string{
				common.NsmServerSocketEnv: containerServerSocketFile(deviceid),
				common.NsmClientSocketEnv: containerClientSocketFile(deviceid),
				common.WorkspaceEnv:       containerDeviceDirectory(deviceid),
			}
			if !n.insecure {
				mounts = append(mounts, &pluginapi.Mount{
					ContainerPath: SpireSocket,
					HostPath:      SpireSocket,
					ReadOnly:      true,
				})
			}
			if n.insecure {
				envs[tools.InsecureEnv] = strconv.FormatBool(true)
			}
			containerResponse := &pluginapi.ContainerAllocateResponse{
				Envs:   envs,
				Mounts: mounts,
			}
			containerResponses = append(containerResponses, containerResponse)
			n.executor.Exec(func() {
				n.allocatedDevices[deviceid] = n.devices[deviceid]
			})
		}
	}
	n.resizeDevicePool()
	return &pluginapi.AllocateResponse{ContainerResponses: containerResponses}, nil
}

func (n *nsmgrDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (n *nsmgrDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (n *nsmgrDevicePlugin) resizeDevicePool() {
	n.executor.Exec(func() {
		for len(n.devices)-len(n.allocatedDevices) < DeviceBuffer {
			device := &pluginapi.Device{
				ID:     "nsm-" + string(len(n.devices)),
				Health: pluginapi.Healthy,
			}
			listener, err := net.Listen("unix", localServerSocketFile(device.GetID()))
			if err != nil {
				// Note: There's nothing productive we can do about this other than failing here
				// and thus not increasing the device pool
				return
			}
			grpcServer := grpc.NewServer()
			go func() {
				grpcServer.Serve(listener)
			}()
			n.Nsmgr.Register(grpcServer)
			n.devices[device.GetID()] = device
		}
		listAndWatchResponse := &pluginapi.ListAndWatchResponse{}
		for _, device := range n.devices {
			listAndWatchResponse.Devices = append(listAndWatchResponse.Devices, device)
		}
		for _, listAndWatchListener := range n.listAndWatchListeners {
			listAndWatchListener.Send(listAndWatchResponse)
		}
	})
}
