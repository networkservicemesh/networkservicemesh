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
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nseconnect"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/ligato/networkservicemesh/plugins/nsmserver"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// clientConnectionTimeout defines time the client waits for establishing connection with the server
	clientConnectionTimeout = time.Second * 60
	// networkServiceName defines Network Service Name the NSE is serving for
	networkServiceName = "gold-network"
	// location of network namespace for a process
	netnsfile = "/proc/self/ns/net"
	// MaxSymLink is maximum length of Symbolic Link
	MaxSymLink = 8192
)

var (
	clientSocketPath     = path.Join(nsmserver.SocketBaseDir, nsmserver.ServerSock)
	clientSocketUserPath = flag.String("nsm-socket", "", "Location of NSM process client access socket")
	nseSocketName        = flag.String("nse-socket", "nse.ligato.io.sock", "Name of NSE socket whcih will be used by NSM for Connection Request call")
	kubeconfig           = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
)

type nseConnection struct {
	metadata           *common.Metadata
	podUID             string
	networkServiceName string
	linuxNamespace     string
}

func (n nseConnection) RequestNSEConnection(ctx context.Context, req *nseconnect.NSEConnectionRequest) (*nseconnect.NSEConnectionReply, error) {

	return &nseconnect.NSEConnectionReply{
		RequestId:          n.podUID,
		Metadata:           n.metadata,
		NetworkServiceName: n.networkServiceName,
		LinuxNamespace:     n.linuxNamespace,
	}, nil
}

func dial(ctx context.Context, unixSocketPath string) (*grpc.ClientConn, error) {
	c, err := grpc.DialContext(ctx, unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	return c, err
}

func buildClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	kubeconfigEnv := os.Getenv("KUBECONFIG")

	if kubeconfigEnv != "" {
		kubeconfig = &kubeconfigEnv
	}

	if *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, err
	}
	k8s, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return k8s, nil
}

func main() {
	// TODO (sbezverk) migtate to cobra package for flags and arguments
	flag.Parse()
	var wg sync.WaitGroup

	k8s, err := buildClient()
	if err != nil {
		logrus.Errorf("nsm client: fail to build kubernetes client with error: %+v, exiting...", err)
		os.Exit(1)
	}
	namespace := os.Getenv("INIT_NAMESPACE")
	if namespace == "" {
		logrus.Error("nsm client: cannot detect namespace, make sure INIT_NAMESPACE variable is set via downward api, exiting...")
		os.Exit(1)
	}
	podName := os.Getenv("HOSTNAME")

	// podUID is used as a unique identifier for nsm init process, it will stay the same throughout life of
	// pod and will guarantee idempotency of possible repeated requests to NSM
	pod, err := k8s.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("nsm client: failure to get pod  %s/%s with error: %+v, exiting...", namespace, podName, err)
		os.Exit(1)
	}
	podUID := string(pod.GetUID())

	// For NSE to program container's dataplane, container's linux namespace must be sent to NSM
	linuxNS := getCurrentNS()
	if linuxNS == "" {
		logrus.Fatal("nsm client: failed to get a linux namespace for the current process, exiting...")
		os.Exit(1)
	}

	// Checking if nsm client socket exists and of not crash NSE
	clientSocket := clientSocketPath
	if clientSocketUserPath != nil {
		clientSocket = *clientSocketUserPath
	}

	if _, err := os.Stat(clientSocket); err != nil {
		logrus.Fatalf("nse: failure to access nsm socket at %s with error: %+v, exiting...", clientSocket, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), clientConnectionTimeout)
	defer cancel()
	conn, err := dial(ctx, clientSocket)
	if err != nil {
		logrus.Fatalf("nse: failure to communicate with the socket %s with error: %+v", clientSocket, err)
	}
	defer conn.Close()
	logrus.Infof("nse: connection to nsm server on socket: %s succeeded.", clientSocket)

	// NSM socket path will be used to drop NSE socket for NSM's Connection request
	nsePath, _ := filepath.Split(clientSocket)
	if err := socketCleanup(path.Join(nsePath, *nseSocketName)); err != nil {
		logrus.Fatalf("nse: failure to cleanup stale socket %s with error: %+v", path.Join(nsePath, *nseSocketName), err)
	}
	nse, err := net.Listen("unix", path.Join(nsePath, *nseSocketName))
	grpcServer := grpc.NewServer()

	// Registering NSE API, it will listen for Connection requests from NSM and return information
	// needed for NSE's dataplane programming.
	nseConn := nseConnection{
		metadata: &common.Metadata{
			Name:      podName,
			Namespace: namespace,
		},
		podUID:             podUID,
		networkServiceName: networkServiceName,
		linuxNamespace:     linuxNS,
	}

	nseconnect.RegisterNSEConnectionServer(grpcServer, nseConn)
	go func() {
		wg.Add(1)
		if err := grpcServer.Serve(nse); err != nil {
			logrus.Fatalf("nse: failed to start grpc server on socket %s with error: %+v ", path.Join(nsePath, *nseSocketName), err)
		}
	}()
	// Check if the socket of device plugin server is operation
	if err := socketOperationCheck(path.Join(nsePath, *nseSocketName)); err != nil {
		logrus.Fatalf("nse: failure to communicate with the socket %s with error: %+v", path.Join(nsePath, *nseSocketName), err)
	}

	// Ok, NSE server is ready and now the channel can be advertised to NSM
	nsmClient := nsmconnect.NewClientConnectionClient(conn)

	channel := netmesh.NetworkServiceChannel{
		Metadata: &common.Metadata{
			Name: "gold-net-channel-1",
		},
		NetworkServiceName: networkServiceName,
		Payload:            "ipv4",
		SocketLocation:     path.Join(nsePath, *nseSocketName),
		Interface: []*common.Interface{
			{
				Type: common.InterfaceType_KERNEL_INTERFACE,
				Metadata: &common.Metadata{
					Name: "kernel_interface_1",
				},
				Preference: common.InterfacePreference_FIRST,
			},
		},
	}
	channels := make([]*netmesh.NetworkServiceChannel, 0)
	channels = append(channels, &channel)
	resp, err := nsmClient.RequestAdvertiseChannel(context.Background(), &nsmconnect.ChannelAdvertiseRequest{
		NetmeshChannel: channels,
	})
	if err != nil {
		grpcServer.Stop()
		logrus.Fatalf("nse: failure to communicate with the socket %s with error: %+v", clientSocket, err)

	}
	if !resp.Success {
		grpcServer.Stop()
		logrus.Fatalf("nse: NSM response is inidcating failure of accepting Channel Advertisiment.")
	}

	logrus.Infof("nse: channel has been successfully advertised, waiting for connection from NSM...")
	// Now block on WaitGroup
	wg.Wait()
}

func socketOperationCheck(listenEndpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := dial(ctx, listenEndpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	return nil
}

func socketCleanup(listenEndpoint string) error {
	fi, err := os.Stat(listenEndpoint)
	if err == nil && (fi.Mode()&os.ModeSocket) != 0 {
		if err := os.Remove(listenEndpoint); err != nil {
			return fmt.Errorf("cannot remove listen endpoint %s with error: %+v", listenEndpoint, err)
		}
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failure stat of socket file %s with error: %+v", listenEndpoint, err)
	}
	return nil
}

func getCurrentNS() string {
	buf := make([]byte, MaxSymLink)
	numBytes, err := syscall.Readlink(netnsfile, buf)
	if err != nil {
		return ""
	}
	link := string(buf[0:numBytes])
	nsRegExp := regexp.MustCompile("net:\\[(.*)\\]")
	submatches := nsRegExp.FindStringSubmatch(link)
	if len(submatches) >= 1 {
		return submatches[1]
	}
	return ""
}
