package main

import (
	"flag"
	"github.com/ligato/networkservicemesh/k8s/pkg/apis/networkservice/v1"
	"github.com/ligato/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"reflect"
)

func main() {
	address := "127.0.0.1:5000"
	logrus.Println("Starting NSMD Kubernetes on " + address)

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

	clientset, e := clientset.NewForConfig(config)
	if e != nil {
		logrus.Fatalln("Unable to initialize nsmd-k8s", e)
	}

	names := v1beta1.CustomResourceDefinitionNames{
		Plural:     "networkservices",
		Singular:   "networkservice",
		ShortNames: []string{"netsvc", "netsvcs"},
		Kind:       reflect.TypeOf(v1.NetworkService{}).Name(),
	}

	svc := v1.NetworkService{}

	err = CreateCRD("networkservices.networkservicemesh.io", "networkservicemesh.io", "v1", v1beta1.ClusterScoped, names, svc, clientset)
	if err != nil {
		logrus.Fatalln(err)
	}

	nsmClientSet, err := versioned.NewForConfig(config)
	_, err = nsmClientSet.Networkservicemesh().NetworkServices("default").Create(&v1.NetworkService{
		ObjectMeta: v12.ObjectMeta{
			Name: "secure-intranet-connectivity",
		},
		Spec: v1.NetworkServiceSpec{
			Payload: "IP",
		},
		Status: v1.NetworkServiceStatus{},
	})

	if err != nil {
		logrus.Fatalln(err)
	}

	// Start NSC Client

	// Start InterNSM

}

// Create the CRD resource, ignore error if it already exists
func CreateCRD(name, group, version string, scope v1beta1.ResourceScope, names v1beta1.CustomResourceDefinitionNames, netsvc v1.NetworkService, clientset *clientset.Clientset) error {
	crd := &v1beta1.CustomResourceDefinition{
		ObjectMeta: v12.ObjectMeta{Name: name},
		Spec: v1beta1.CustomResourceDefinitionSpec{
			Group:   group,
			Version: version,
			Scope:   scope,
			Names:   names,
		},
	}

	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil && apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
