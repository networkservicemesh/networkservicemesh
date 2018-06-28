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

	"github.com/vishvananda/netns"

	"github.com/ligato/networkservicemesh/nsmdp"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
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
)

var (
	clientSocketPath     = path.Join(nsmdp.SocketBaseDir, nsmdp.ServerSock)
	clientSocketUserPath = flag.String("nsm-socket", "", "Location of NSM process client access socket")
	configMapName        = flag.String("configmap-name", "", "Name of a ConfigMap with requested configuration.")
	kubeconfig           = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
)

type networkService struct {
	Name             string                 `json:"name" yaml:"name"`
	ServiceInterface []nsmconnect.Interface `json:"serviceInterface" yaml:"serviceInterface"`
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

func applyRequiredConfig(ns []*networkService) error {
	currentNamespace, err := netns.Get()
	if err != nil {
		logrus.Errorf("nsm client: failure to get pod's namespace with error: %+v", err)
		os.Exit(1)
	}
	logrus.Infof("nsm client: pod's namespace is [%s]", currentNamespace.String())
	namespaceHandle, err := netlink.NewHandleAt(currentNamespace)
	if err != nil {
		logrus.Errorf("nsm client: failure to get pod's handle with error: %+v", err)
		os.Exit(1)
	}
	interfaces, err := namespaceHandle.LinkList()
	if err != nil {
		logrus.Errorf("nsm client: pailure to get pod's interfaces with error: %+v", err)
	}
	logrus.Info("nsm client: pod's interfaces:")
	for _, intf := range interfaces {
		logrus.Infof("Name: %s Type: %s", intf.Attrs().Name, intf.Type())
	}
	logrus.Info("nsm client: requested network services:")
	for _, s := range ns {
		logrus.Infof("%+v", s)
	}
	return nil
}

func parseConfigMap(cm *v1.ConfigMap) ([]*networkService, error) {
	nSs := []*networkService{}
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
	conn, err := dial(ctx, clientSocket)
	if err != nil {
		logrus.Errorf("nsm client: failure to communicate with the socket %s with error: %+v", clientSocket, err)
		os.Exit(1)
	}
	nsmClient := nsmconnect.NewClientConnectionClient(conn)
	defer conn.Close()
	defer cancel()
	logrus.Infof("nsm client: connection to nsm server on socket: %s succeeded.", clientSocket)
	logrus.Infof("nsm client: client api %+v", nsmClient)
	// Init related activities start here

	if err := applyRequiredConfig(ns); err != nil {
		logrus.Infof("nsm client: initialization failed, exiting...", err)
		os.Exit(1)
	}
	// Init related activities ends here
	logrus.Info("nsm client: initialization is completed successfully, exiting...")
	os.Exit(0)
}
