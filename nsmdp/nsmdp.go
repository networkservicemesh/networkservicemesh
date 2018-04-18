package nsmdp

import (
	"github.com/ligato/networkservicemesh/deviceplugin"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type NSMDevicePlugin struct {
	*deviceplugin.DevicePlugin
}

type NSMDevice struct {
	*pluginapi.Device
	Token string
}

const (
	resourceName    = "nsm.ligato.io"
	serverSock      = pluginapi.DevicePluginPath + "nsm.ligato.io.sock"
	initDeviceCount = 10
)

//func (n *NSMDevicePlugin) NewNSMDevicePlugin() *NSMDevicePlugin {
//	n.NewDevicePlugin(serverSock,resourceName)
//	for i:=uint(0);i<initDeviceCount;i++ {
//		devs = append(n.devs,&NSMDevice{
//			ID: "NSM-" + string(i),
//			Health: pluginapi.Healthy,
////			Token: generateToken(),
//		}
//	}
//}

/*
func (n *NSMDevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	return nil,nil;
}

func (n *NSMDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	for {
		select {
		case <-n.stop:
			return nil;
		case dev :=<-d.health;
	}
}

*/
