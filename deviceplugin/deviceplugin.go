// Package deviceplugin provides a basic implementation of a Kubernetes DevicePlugin without content.
// It is intended to be used to build other DevicePlugins.
package deviceplugin

import (
	"log"
	"net"
	"os"
	//      "strings"
	"time"
	"path"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

type DevicePlugin struct {
	socket string;
	resourceName string;
	stop chan interface {};
	server *grpc.Server;
}

func NewDevicePlugin(serversock string, resourcename string) *DevicePlugin {
	return &DevicePlugin{
		socket: serversock,
		stop: make(chan interface{}),
		resourceName: resourcename,
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

func (d *DevicePlugin) cleanup() error {
	if err := os.Remove(d.socket); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (d *DevicePlugin) Start() error {
	err := d.cleanup();

	if err != nil {
		return err;
	}

	sock,err := net.Listen("unix",d.socket);
	if err != nil {
		return err
	}
	d.server = grpc.NewServer([]grpc.ServerOption{}...);
	pluginapi.RegisterDevicePluginServer(d.server,d);

	go d.server.Serve(sock);

	conn,err := dial(d.socket, 5*time.Second);
	if err != nil {
		return err;
	}
	conn.Close();

	return nil;
}

func (d *DevicePlugin) Stop() error {
	if d.server == nil {
		return nil;
	}

	d.server.Stop();
	d.server = nil;
	close (d.stop)
	return d.cleanup();
}

func (d *DevicePlugin) Serve() error {
	err := d.Start();
	if err != nil {
		log.Printf("Could not start device plugin %s",err);
	}
	log.Println("Starting to serve on", d.socket);
	
	err = d.Register(pluginapi.KubeletSocket,d.resourceName);
	if err != nil {
		log.Printf("Could not register device plugin %s", err);
		return err
	}
	log.Println("Registered device plugin with Kubelet");
	return nil;
}

// Define functions needed to meet the Kubernetes DevicePlugin API

func (d *DevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

func (d *DevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	return nil,nil;
}

func (d *DevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	return nil;
}

func (d *DevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (d *DevicePlugin) Register(kubeletEndpoint,resourceName string) error {
	conn,err := dial(kubeletEndpoint, 5*time.Second);
	if err != nil {
		return err;
	}
	defer conn.Close();
	client := pluginapi.NewRegistrationClient(conn);
	request := &pluginapi.RegisterRequest{
		Version: pluginapi.Version,
		Endpoint: path.Base(d.socket),
		ResourceName: resourceName,
	}
	_, err = client.Register(context.Background(),request);
	if err != nil {
		return err;
	}
	return nil;
}

