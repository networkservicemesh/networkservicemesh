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
	"strings"
	"time"

	"regexp"
	"syscall"

	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/ligato/networkservicemesh/plugins/nsmserver"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// clientConnectionTimeout defines time the client waits for establishing connection with the server
	clientConnectionTimeout = time.Second * 60
	// clientConnectionTimeout defines retry interval for establishing connection with the server
	clientConnectionRetry = time.Second * 2
	// location of network namespace for a process
	netnsfile = "/proc/self/ns/net"
	// MaxSymLink is maximum length of Symbolic Link
	MaxSymLink = 8192
)

var (
	clientSocketPath     = path.Join(nsmserver.SocketBaseDir, nsmserver.ServerSock)
	clientSocketUserPath = flag.String("nsm-socket", "", "Location of NSM process client access socket")
	configMapName        = flag.String("configmap-name", "", "Name of a ConfigMap with requested configuration.")
	kubeconfig           = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
)

type networkService struct {
	Name             string              `json:"name" yaml:"name"`
	ServiceInterface []*common.Interface `json:"serviceInterface" yaml:"serviceInterface"`
}

func dial(ctx context.Context, unixSocketPath string) (*grpc.ClientConn, error) {
	c, err := grpc.DialContext(ctx, unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	return c, err
}

func checkClientConfigMap(name, namespace string, k8s kubernetes.Interface) (*v1.ConfigMap, error) {
	return k8s.CoreV1().ConfigMaps(namespace).Get(name, metav1.GetOptions{})
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

func parseConfigMap(cm *v1.ConfigMap) ([]*networkService, error) {
	nSs := make([]*networkService, 0)
	rawData, ok := cm.Data["networkService"]
	if !ok {
		return nil, fmt.Errorf("missing required key 'networkService:'")
	}
	sr := strings.NewReader(rawData)
	decoder := yaml.NewYAMLOrJSONDecoder(sr, 512)
	if err := decoder.Decode(&nSs); err != nil {
		logrus.Errorf("decoding %+v failed with error: %v", rawData, err)
		return nil, err
	}

	return nSs, nil
}

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")

	// Building kubernetes client
	k8s, err := buildClient()
	if err != nil {
		logrus.Errorf("nsm client: fail to build kubernetes client with error: %+v, exiting...", err)
		os.Exit(1)
	}

	// Checking presence of client's ConfigMap, if it is not provided, then init container gracefully exits
	if *configMapName == "" {
		logrus.Info("nsm client: no client's configmap name was provided, exiting...")
		os.Exit(0)
	}
	name := *configMapName
	// POD's namespace is taken from env variable, which must be either defined via downward api for in-cluster case
	// or explicitely exported for out-of-cluster case.
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
	configMap, err := checkClientConfigMap(name, namespace, k8s)
	if err != nil {
		logrus.Errorf("nsm client: failure to access client's configmap at %s/%s with error: %+v, exiting...", namespace, name, err)
		os.Exit(1)
	}
	// Attempting to extract Client's config from the config map and store it in networkService slice
	ns, err := parseConfigMap(configMap)
	if err != nil {
		logrus.Errorf("nsm client: failure to parse client's configmap %s/%s with error: %+v, exiting...", namespace, name, err)
		os.Exit(1)
	}
	// Checking number of NetworkServices extracted from the config map and if it is 0 then
	// print log message and gracefully exit
	if len(ns) == 0 {
		logrus.Infof("nsm client: no NetworkServices were discovered in client's configmap %s/%s, exiting...", namespace, name)
		os.Exit(0)
	}
	// Checking if nsm client socket exists and of not crash init container
	clientSocket := clientSocketPath
	if clientSocketUserPath != nil {
		clientSocket = *clientSocketUserPath
	}

	if _, err := os.Stat(clientSocket); err != nil {
		logrus.Errorf("nsm client: failure to access nsm socket at %s with error: %+v, exiting...", clientSocket, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), clientConnectionTimeout)
	defer cancel()
	conn, err := dial(ctx, clientSocket)
	if err != nil {
		logrus.Errorf("nsm client: failure to communicate with the socket %s with error: %+v", clientSocket, err)
		os.Exit(1)
	}
	defer conn.Close()
	logrus.Infof("nsm client: connection to nsm server on socket: %s succeeded.", clientSocket)

	// Init related activities start here
	nsmClient := nsmconnect.NewClientConnectionClient(conn)

	// Getting list of available NetworkServices on local NSM server
	availablaNetworkServices, err := getNetworkServices(nsmClient)
	if err != nil {
		logrus.Fatalf("nsm client: failed to get a list of NetworkServices from NSM with error: %+v, exiting...", err)
		os.Exit(1)
	}
	if len(availablaNetworkServices) == 0 {
		// Since local NSM has no any NetworkServices, then there is nothing to configure for the client
		logrus.Info("nsm client: Local NSM does not have any NetworkServices, exiting...")
		os.Exit(0)
	}
	logrus.Info("nsm client: list of discovered network services:")
	for _, s := range availablaNetworkServices {
		logrus.Infof("      network service: %s/%s", s.Metadata.Namespace, s.Metadata.Name)
		for _, c := range s.Channel {
			logrus.Infof("            Channel: %s/%s", c.Metadata.Namespace, c.Metadata.Name)
			for _, i := range c.Interface {
				logrus.Infof("                  Interface type: %s preference: %s", i.GetType(), i.GetPreference())
			}
		}
	}
	logrus.Infof("nsm client: %d NetworkServices discovered from Local NSM.", len(availablaNetworkServices))

	// For NSM to program container's dataplane, container's linux namespace must be sent to NSM
	linuxNS, err := getCurrentNS()
	if err != nil {
		logrus.Fatalf("nsm client: failed to get a linux namespace for pod %s/%s with error: %+v, exiting...", namespace, podName, err)
		os.Exit(1)
	}

	// TODO (sbezverk) At this point it is not clear the logic what to do if nsm init process is requesting
	// connection to multiple NetworkServices. Loop through and call Connect for individual? what if one of them
	// fails? Fail all? If one of fails, how to notify NSM to release Connections which succeeded.
	// For now connection to a single NetworkService is requested.
	cReq := nsmconnect.ConnectionRequest{
		RequestId: podUID,
		Metadata: &common.Metadata{
			Name:      podName,
			Namespace: namespace,
		},
		NetworkServiceName: ns[0].Name,
		LinuxNamespace:     linuxNS,
		Interface:          ns[0].ServiceInterface,
	}

	logrus.Infof("Connection request: %+v number of interfaces: %d", cReq, len(cReq.Interface))
	connParams, err := requestConnection(nsmClient, &cReq)
	if err != nil {
		logrus.Fatalf("nsm client: failed to request connection from NSM with error: %+v, exiting...", err)
		os.Exit(1)
	}

	// Init related activities ends here
	logrus.Infof("nsm client: initialization is completed successfully, connection parameters: %+v, exiting...", connParams)
}

func getNetworkServices(nsmClient nsmconnect.ClientConnectionClient) ([]*netmesh.NetworkService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clientConnectionTimeout)
	defer cancel()
	ticker := time.NewTicker(clientConnectionRetry)
	defer ticker.Stop()
	var err error
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("nsm client: Discovery request did not succeed within %d seconds, last known error: %+v", clientConnectionTimeout, err)
		case <-ticker.C:
			resp, err := nsmClient.RequestDiscovery(ctx, &nsmconnect.DiscoveryRequest{})
			if err == nil {
				return resp.NetworkService, nil
			}
			logrus.Infof("nsm client: Discovery request failed with: %+v, re-attempting in %d seconds", err, clientConnectionRetry)
		}
	}
}

func requestConnection(nsmClient nsmconnect.ClientConnectionClient, cReq *nsmconnect.ConnectionRequest) (*nsmconnect.ConnectionParameters, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clientConnectionTimeout)
	defer cancel()
	ticker := time.NewTicker(clientConnectionRetry)
	defer ticker.Stop()
	var err error
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("nsm client: Connection request did not succeed within %d seconds, last known error: %+v", clientConnectionTimeout, err)
		case <-ticker.C:
			cResp, err := nsmClient.RequestConnection(ctx, cReq)
			if err == nil && cResp.Accepted {
				return cResp.ConnectionParameters, nil
			}
			logrus.Infof("nsm client: Connection request failed with: %+v, re-attempting in %d seconds",
				err, clientConnectionRetry)
			if cResp != nil {
				logrus.Infof("nsm client: server side admission response: %s", cResp.AdmissionError)
			}
		}
	}
}

func getCurrentNS() (string, error) {
	buf := make([]byte, MaxSymLink)
	numBytes, err := syscall.Readlink(netnsfile, buf)
	if err != nil {
		return "", err
	}
	link := string(buf[0:numBytes])
	nsRegExp := regexp.MustCompile("net:\\[(.*)\\]")
	submatches := nsRegExp.FindStringSubmatch(link)
	if len(submatches) >= 1 {
		return submatches[1], nil
	}
	return "", fmt.Errorf("namespace is not found")
}
