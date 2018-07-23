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
	"net"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nseconnect"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/ligato/networkservicemesh/pkg/tools"
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
		logrus.Errorf("nse: fail to build kubernetes client with error: %+v, exiting...", err)
		os.Exit(1)
	}
	namespace := os.Getenv("INIT_NAMESPACE")
	if namespace == "" {
		logrus.Error("nse: cannot detect namespace, make sure INIT_NAMESPACE variable is set via downward api, exiting...")
		os.Exit(1)
	}
	podName := os.Getenv("HOSTNAME")

	// podUID is used as a unique identifier for nse init process, it will stay the same throughout life of
	// pod and will guarantee idempotency of possible repeated requests to NSM
	pod, err := k8s.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("nse: failure to get pod  %s/%s with error: %+v, exiting...", namespace, podName, err)
		os.Exit(1)
	}
	podUID := string(pod.GetUID())

	// For NSE to program container's dataplane, container's linux namespace must be sent to NSM
	linuxNS, err := tools.GetCurrentNS()
	if err != nil {
		logrus.Fatalf("nse: failed to get a linux namespace for pod %s/%s with error: %+v, exiting...", namespace, podName, err)
		os.Exit(1)
	}

	// Checking if non default location of NSM socket was provided
	clientSocket := clientSocketPath
	if clientSocketUserPath != nil {
		clientSocket = *clientSocketUserPath
	}

	if _, err := os.Stat(clientSocket); err != nil {
		logrus.Fatalf("nse: failure to access nsm socket at %s with error: %+v, exiting...", clientSocket, err)
	}

	conn, err := tools.SocketOperationCheck(clientSocket)
	if err != nil {
		logrus.Fatalf("nse: failure to communicate with nsm on the socket %s with error: %+v", clientSocket, err)
	}
	logrus.Infof("nse: connection to nsm server on the socket: %s succeeded.", clientSocket)
	defer conn.Close()

	// NSM socket path will be used to drop NSE socket for NSM's Connection request
	nsePath, _ := filepath.Split(clientSocket)
	nseSocket := path.Join(nsePath, *nseSocketName)
	if err := tools.SocketCleanup(nseSocket); err != nil {
		logrus.Fatalf("nse: failure to cleanup stale socket %s with error: %+v", nseSocket, err)
	}
	nse, err := net.Listen("unix", nseSocket)
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
			logrus.Fatalf("nse: failed to start grpc server on socket %s with error: %+v ", nseSocket, err)
		}
	}()
	// Check if the socket of device plugin server is operation
	testSocket, err := tools.SocketOperationCheck(nseSocket)
	if err != nil {
		logrus.Fatalf("nse: failure to communicate with the socket %s with error: %+v", nseSocket, err)
	}
	testSocket.Close()

	// Ok, NSE server is ready and now the channel can be advertised to NSM
	nsmClient := nsmconnect.NewClientConnectionClient(conn)

	channel := netmesh.NetworkServiceChannel{
		Metadata: &common.Metadata{
			Name: "gold-net-channel-1",
		},
		NseProviderName:    podName,
		NetworkServiceName: networkServiceName,
		Payload:            "ipv4",
		SocketLocation:     nseSocket,
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
