package utils

import (
	"flag"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/clientset/versioned"
)

// NewClientSet creates a new Clientset for the default kubernetes config.
func NewClientSet() (*versioned.Clientset, *rest.Config, error) {
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
			return nil, nil, errors.Wrap(err, "unable to build config")
		}
	}

	nsmClientSet, err := versioned.NewForConfig(config)
	return nsmClientSet, config, err
}
