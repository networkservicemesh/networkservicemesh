package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/ligato/networkservicemesh/controlplane/pkg/apis/crossconnect"
	"github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/ligato/networkservicemesh/pkg/tools"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var closing = false
var managers = map[string]string{}

func monitorCrossConnects(address string, continuousMonitor bool) {
	var err error
	logrus.Infof("Starting CrossConnections Monitor on %s", address)
	tracer := opentracing.GlobalTracer()
	conn, err := grpc.Dial(address, grpc.WithInsecure(),
		grpc.WithStreamInterceptor(
			otgrpc.OpenTracingStreamClientInterceptor(tracer)),
		grpc.WithUnaryInterceptor(
			otgrpc.OpenTracingClientInterceptor(tracer, otgrpc.LogPayloads())))
	if err != nil {
		logrus.Errorf("failure to communicate with the socket %s with error: %+v", address, err)
		return
	}
	defer conn.Close()
	dataplaneClient := crossconnect.NewMonitorCrossConnectClient(conn)

	// Looping indefinetly or until grpc returns an error indicating the other end closed connection.
	stream, err := dataplaneClient.MonitorCrossConnects(context.Background(), &empty.Empty{})

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
		sp := tracer.StartSpan("cross connect event")
		sp.Finish()
		data := ""
		println("\u001b[0m\n")
		for id, cc := range event.CrossConnects {
			if cc == nil {
				data += fmt.Sprintf("\u001b[31m*** %s %s Id:%s\n\u001b[0m", address, event.Type, id)
			} else {
				data += fmt.Sprintf("\u001b[31m*** %s %s Id:%s \n\u001b[32m%s\n\u001b[0m", address, event.Type, id, t.Text(cc))
			}
		}
		println(data)
		if !continuousMonitor {
			logrus.Infof("Monitoring of server: %s. is complete...", address)
			delete(managers, address)
			return
		}
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

	for !closing {
		result, err := nsmClientSet.Networkservicemesh().NetworkServiceManagers("default").List(metav1.ListOptions{})
		if err != nil {
			logrus.Fatalln("Unable to find NSMs", err)
		}
		for _, mgr := range result.Items {
			if _, ok := managers[mgr.Status.URL]; !ok {
				logrus.Printf("Found manager: %s at %s", mgr.Name, mgr.Status.URL)
				managers[mgr.Status.URL] = "true"
				go monitorCrossConnects(mgr.Status.URL, true)
			}
		}
		time.Sleep(time.Second)
	}
}

func main() {
	tracer, closer := tools.InitJaeger("crossconnect-monitor")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		closing = true
		wg.Done()
	}()

	lookForNSMServers()

	wg.Wait()

}
