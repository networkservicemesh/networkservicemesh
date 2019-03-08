package main

import (
	"flag"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/registryserver"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// Capture signals to cleanup before exiting
	c := make(chan os.Signal, 1)
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	tracer, closer := tools.InitJaeger("nsmd-k8s")
	opentracing.SetGlobalTracer(tracer)
	defer closer.Close()

	address := os.Getenv("NSMD_K8S_ADDRESS")
	if strings.TrimSpace(address) == "" {
		address = "127.0.0.1:5000"
	}
	nsmName, ok := os.LookupEnv("NODE_NAME")
	if !ok {
		logrus.Fatalf("You must set env variable NODE_NAME to match the name of your Node.  See https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/")
	}
	logrus.Println("Starting NSMD Kubernetes on " + address + " with NsmName " + nsmName)

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

	nsmClientSet, err := versioned.NewForConfig(config)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Fatalln(err)
	}

	server := registryserver.New(nsmClientSet, nsmName)

	logrus.Print("nsmd-k8s initialized and waiting for connection")
	err = server.Serve(listener)
	logrus.Fatalln(err)
	<-c
}
