
package main

import (
	"log"
	"net"
	"os"
//	"strings"
	"time"
	"path"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	resourceName = "nsm.ligato.io"
	serverSock = pluginapi.DevicePluginPath + "nsm.sock";
)

type NetworkServiceManagerDevicePlugin struct {
	socket string;
	stop chan interface {};
	server *grpc.Server;
}

func NewNetworkServiceManagerDevicePlugin() *NetworkServiceManagerDevicePlugin {
	return &NetworkServiceManagerDevicePlugin {
		socket: serverSock,
		stop: make(chan interface{}),
	}
}

func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(),grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string,timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout);
		}),
	)

	if err != nil {
		return nil, err;
	}

	return c, nil;

}

func (nsm *NetworkServiceManagerDevicePlugin) Start() error {
	err := nsm.cleanup();

	if err != nil {
		return err;
	}
	
	sock,err := net.Listen("unix",nsm.socket);
	if err != nil {
		return err
	}
	nsm.server = grpc.NewServer([]grpc.ServerOption{}...);
	pluginapi.RegisterDevicePluginServer(nsm.server,nsm);

	go nsm.server.Serve(sock);

	conn,err := dial(nsm.socket, 5*time.Second);
	if err != nil {
		return err;
	}
	conn.Close();

	return nil;
}

func (nsm *NetworkServiceManagerDevicePlugin) Stop() error {
	if nsm.server == nil {
		return nil;
	}

	nsm.server.Stop();
	nsm.server = nil;
	close (nsm.stop)
	return nsm.cleanup();
}

func (nsm *NetworkServiceManagerDevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	return nil,nil;
}

func (nsm *NetworkServiceManagerDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (nsm *NetworkServiceManagerDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	return nil;
}

func (nsm *NetworkServiceManagerDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (nsm *NetworkServiceManagerDevicePlugin) cleanup() error {
	if err := os.Remove(nsm.socket); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (nsm *NetworkServiceManagerDevicePlugin) Register(kubeletEndpoint, resourceName string) error {
	conn,err := dial(kubeletEndpoint, 5*time.Second);
	if err != nil {
		return err;
	}
	defer conn.Close();
	client := pluginapi.NewRegistrationClient(conn);
	request := &pluginapi.RegisterRequest{
		Version: pluginapi.Version,
		Endpoint: path.Base(nsm.socket),
		ResourceName: resourceName,
	}
	_, err = client.Register(context.Background(),request);
	if err != nil {
		return err;
	}
	return nil;
}

func (nsm *NetworkServiceManagerDevicePlugin) Serve() error {
	err := nsm.Register(pluginapi.KubeletSocket,resourceName);
	if err != nil {
		log.Printf("Could not register device plugin %s", err);
		return err
	}
	log.Println("Registered device plugin with Kubelet");
	return nil;
}


