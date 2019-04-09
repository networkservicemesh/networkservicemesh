package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/apis/crossconnect"
	v1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/informers/externalversions"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver/resource_cache"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Dict to map nsm URL to grpc connection
var managers = map[string]*grpc.ClientConn{}
var stopInformer func()

func deleteManager(manager string) {
	conn, ok := managers[manager]
	if !ok {
		logrus.Warningf("Error: no connection to manager %s", manager)
		return
	}
	conn.Close()
	delete(managers, manager)
}

func monitorCrossConnects(manager string) {
	var err error
	logrus.Infof("Starting CrossConnections Monitor on %s", manager)

	conn, ok := managers[manager]
	if !ok {
		logrus.Warningf("Error: no connection to manager %s", manager)
		return
	}
	defer deleteManager(manager)
	nsmClient := crossconnect.NewMonitorCrossConnectClient(conn)

	// Looping indefinetly or until grpc returns an error indicating the other end closed connection.
	stream, err := nsmClient.MonitorCrossConnects(context.Background(), &empty.Empty{})

	if err != nil {
		logrus.Warningf("Error: %+v.", err)
		return
	}
	t := proto.TextMarshaler{}
	for {
		event, err := stream.Recv()
		if err != nil {
			logrus.Errorf("Error: %+v.", err)
			return
		}
		data := fmt.Sprintf("\u001b[31m*** %s\n\u001b[0m", event.Type)
		data += fmt.Sprintf("\u001b[31m*** %s\n\u001b[0m", conn.Target())
		for _, cc := range event.CrossConnects {
			if cc != nil {
				data += fmt.Sprintf("\u001b[32m%s\n\u001b[0m", t.Text(cc))
			}
		}
		println(data)
	}
}

func lookForNSMServers() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// check if CRD is installed
	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.Println("Unable to get in cluster config, attempting to fall back to kubeconfig", err)
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			logrus.Fatalln("Unable to build config", err)
		}
	}

	// Initialize clientset
	nsmClientSet, err := versioned.NewForConfig(config)
	if err != nil {
		logrus.Fatalln("Unable to initialize nsmd-k8s", err)
	}
	// register the cacheInformer
	nsmCache := resource_cache.NewNetworkServiceManagerCache()
	nsmCache.AddedHandler = nsmGrAdded

	factory := externalversions.NewSharedInformerFactory(nsmClientSet, 0)
	stopFunc, err := nsmCache.Start(factory)
	if err != nil {
		logrus.Fatalln("Unable to start k8s informer", err)
	}
	stopInformer = stopFunc
}

func connectAndMonitor(nsm *v1.NetworkServiceManager) {
	conn, err := grpc.Dial(nsm.Status.URL, grpc.WithInsecure())
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", nsm.Status.URL, err)
		return
	}
	logrus.Printf("Found manager: %s at %s", nsm.Name, nsm.Status.URL)
	managers[nsm.Name] = conn
	go monitorCrossConnects(nsm.Name)

	return
}

func nsmGrAdded(nsm *v1.NetworkServiceManager) {
	logrus.Infof("nsmgr %v added", nsm.Name)
	if _, ok := managers[nsm.Name]; !ok {
		connectAndMonitor(nsm)
	} else {
		logrus.Warningf("Received nsmGrAdded event for a nsmgr that alredy exists: %v. No new connection will be created", nsm.Name)
	}
	return
}

func main() {

	var wg sync.WaitGroup
	wg.Add(1)

	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()
	go func() {
		<-c
		wg.Done()
	}()

	lookForNSMServers()

	wg.Wait()
	stopInformer()
}
