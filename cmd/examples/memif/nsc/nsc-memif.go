package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"time"

	nsmapi "github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1"
	nsmclient "github.com/ligato/networkservicemesh/pkg/client/clientset/versioned"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/common"
	"github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect"
	"github.com/ligato/networkservicemesh/pkg/tools"
	"github.com/ligato/networkservicemesh/plugins/nsmserver"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// networkServiceName defines Network Service Name the NSM client request endpoint for
	networkServiceName = "memif-test"
)

const (
	// clientConnectionTimeout defines time the client waits for establishing connection with the server
	clientConnectionTimeout = time.Second * 60
	// clientConnectionTimeout defines retry interval for establishing connection with the server
	clientConnectionRetry = time.Second * 2
)

var (
	clientSocketPath     = path.Join(nsmserver.SocketBaseDir, nsmserver.ServerSock)
	clientSocketUserPath = flag.String("nsm-socket", "", "Location of NSM process client access socket")
	kubeconfig           = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Either this or master needs to be set if the provisioner is being run out of cluster.")
)

func main() {
	flag.Parse()
	flag.Set("logtostderr", "true")

	// Building kubernetes client
	k8sClient, nsmClient, err := k8sBuildClient()
	if err != nil {
		logrus.Errorf("nsm client: fail to build kubernetes client with error: %+v, exiting...", err)
		os.Exit(1)
	}

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
	pod, err := k8sClient.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("nsm client: failure to get pod  %s/%s with error: %+v, exiting...", namespace, podName, err)
		os.Exit(1)
	}
	podUID := string(pod.GetUID())

	availableEndpoints, err := getNetworkServiceEndpoint(nsmClient, networkServiceName, namespace)
	if err != nil {
		logrus.Fatalf("nsm client: failed to get a list of NetworkServices from NSM with error: %+v, exiting...", err)
		os.Exit(1)
	}

	for _, endpoint := range availableEndpoints {
		logrus.Infof(" - endpoint: %s", endpoint.ObjectMeta.Name)
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

	conn, err := tools.SocketOperationCheck(clientSocket)
	if err != nil {
		logrus.Fatalf("nsm client: failure to communicate with the socket %s with error: %+v", clientSocket, err)
	}
	defer conn.Close()
	logrus.Infof("nsm client: connection to nsm server on socket: %s succeeded.", clientSocket)

	// Init related activities start here
	nsmConnectionClient := nsmconnect.NewClientConnectionClient(conn)

	// For NSM to program container's dataplane, container's linux namespace must be sent to NSM
	linuxNS, err := tools.GetCurrentNS()
	if err != nil {
		logrus.Fatalf("nsm client: failed to get a linux namespace for pod %s/%s with error: %+v, exiting...", namespace, podName, err)
		os.Exit(1)
	}

	mechanism := &common.LocalMechanism{
		Type: common.LocalMechanismType_KERNEL_INTERFACE,
	}

	cReq := nsmconnect.ConnectionRequest{
		RequestId:          podUID,
		NetworkServiceName: networkServiceName,
		LinuxNamespace:     linuxNS,
		LocalMechanisms:    []*common.LocalMechanism{mechanism},
	}

	logrus.Infof("Connection request: %+v number of interfaces: %d", cReq, len(cReq.LocalMechanisms))
	connParams, err := requestConnection(nsmConnectionClient, &cReq)
	if err != nil {
		logrus.Fatalf("nsm client: failed to request connection for Network Service %s with error: %+v, exiting...", networkServiceName, err)
		os.Exit(1)
	}
	logrus.Infof("nsm client: connection to Network Service %s suceeded, connection parameters: %+v, exiting...", networkServiceName, connParams)
	// Init related activities ends here

	logrus.Info("nsm client: initialization is completed successfully, exiting...")
}

func k8sBuildClient() (*kubernetes.Clientset, *nsmclient.Clientset, error) {
	var config *rest.Config
	var err error

	if *kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create config: %v", err)
	}
	k8s, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client: %v", err)
	}
	customResourceClient, err := nsmclient.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create customresource clientset: %v", err)
	}

	return k8s, customResourceClient, nil
}

func getNetworkServiceEndpoint(
	nsmClient *nsmclient.Clientset,
	networkService string,
	namespace string) ([]nsmapi.NetworkServiceEndpoint, error) {
	selector := labels.SelectorFromSet(labels.Set(map[string]string{nsmserver.EndpointServiceLabel: networkService}))
	options := metav1.ListOptions{LabelSelector: selector.String()}
	endpointList, err := nsmClient.NetworkserviceV1().NetworkServiceEndpoints(namespace).List(options)
	if err != nil {
		return nil, err
	}
	return endpointList.Items, nil
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
			return nil, fmt.Errorf("nsm client: Request Connection to NSM timedout (%d)seconds with error: %+v", clientConnectionTimeout, err)
		case <-ticker.C:
			cResp, err := nsmClient.RequestConnection(ctx, cReq)
			if err == nil && cResp.Accepted {
				return cResp.ConnectionParameters, nil
			}
			switch status.Convert(err).Code() {
			case codes.Aborted:
				// Aborted inidcates an unrecoverable issue, retries are not needed
				fallthrough
			case codes.NotFound:
				// NotFound indicates that requested Network Service does not exist, retries are not needed
				return nil, fmt.Errorf("nsm client: Request Connection to NSM has failed with error: %+v", err)
			case codes.AlreadyExists:
				// AlreadyExists inidcates not completed dataplane programming, will retry until connection timeout expires
				// or success returned
				logrus.Infof("nsm client: NSM inidcates already existing non-completed Connection Request, retrying in %d seconds",
					clientConnectionRetry)
			default:
				logrus.Infof("nsm client: Request Connection to NSM has failed with error: %+v, retrying in %d seconds", err, clientConnectionRetry)
			}
			// There was no error, but NSM did not set Accepted as true, possibly unaccounted error condition or a bug
			if cResp != nil {
				logrus.Infof("nsm client: NSM failed Connection Request with an admission error: %s, check NSM log for more details. Failed request ID: %s",
					cResp.AdmissionError, cReq.RequestId)
			}
		}
	}
}
