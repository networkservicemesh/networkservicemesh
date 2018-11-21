package main

import (
	"flag"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/ligato/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/ligato/networkservicemesh/k8s/pkg/registryserver"
	"github.com/sirupsen/logrus"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"os/signal"
	"syscall"
)

func main() {
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

	// Initialize clientset
	clientset, e := clientset.NewForConfig(config)
	if e != nil {
		logrus.Fatalln("Unable to initialize nsmd-k8s", e)
	}

	err = installCRDs(clientset)

	nsmClientSet, err := versioned.NewForConfig(config)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Fatalln(err)
	}

	server := registryserver.New(nsmClientSet, nsmName)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		removeCRDs(clientset)
	}()

	err = server.Serve(listener)
	logrus.Fatalln(err)
}

var nsmCRDNames = [...]v1beta1.CustomResourceDefinitionNames{
	{
		Plural:     "networkservices",
		Singular:   "networkservice",
		ShortNames: []string{"netsvc", "netsvcs"},
		Kind:       reflect.TypeOf(v1.NetworkService{}).Name(),
	},
	{
		Plural:     "networkserviceendpoints",
		Singular:   "networkserviceendpoint",
		ShortNames: []string{"nse", "nses"},
		Kind:       reflect.TypeOf(v1.NetworkServiceEndpoint{}).Name(),
	},
	{
		Plural:     "networkservicemanagers",
		Singular:   "networkservicemanager",
		ShortNames: []string{"nsm", "nsms"},
		Kind:       reflect.TypeOf(v1.NetworkServiceManager{}).Name(),
	},
}

const nsmGroup = "networkservicemesh.io"

func installCRDs(clientset *clientset.Clientset) error {
	// Install NSM CRDs
	for _, crdName := range nsmCRDNames {
		crd := &v1beta1.CustomResourceDefinition{
			ObjectMeta: v12.ObjectMeta{Name: crdName.Plural + "." + nsmGroup},
			Spec: v1beta1.CustomResourceDefinitionSpec{
				Group:   nsmGroup,
				Version: "v1",
				Scope:   v1beta1.ClusterScoped,
				Names:   crdName,
			},
		}

		logrus.Printf("Registering %s", crd.ObjectMeta.Name)
		_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			logrus.Fatalln(err)
		}
	}
	return nil
}

func removeCRDs(clientset *clientset.Clientset) {
	for _, crdName := range nsmCRDNames {
		name := crdName.Plural + "." + nsmGroup
		logrus.Infof("Deleting resource %s", name)
		err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(name, &v12.DeleteOptions{})
		if err != nil {
			logrus.Errorf("Error %v", err)
		}
	}
}
