package main

import (
	"encoding/json"
	"fmt"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/sirupsen/logrus"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	v1 "k8s.io/kubernetes/pkg/apis/core/v1"
)

const (
	certFile = "/etc/webhook/certs/cert.pem"
	keyFile  = "/etc/webhook/certs/key.pem"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

var (
	deserializer      runtime.Decoder
	ignoredNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}

	repo          string
	initContainer string
	tag           string
)

const (
	nsmAnnotationKey = "ns.networkservicemesh.io"

	repoEnv          = "REPO"
	initContainerEnv = "INITCONTAINER"
	tagEnv           = "TAG"

	repoDefault          = "networkservicemesh"
	initContainerDefault = "nsc"
	tagDefault           = "latest"

	initContainerName = "nsm-init-container"

	pathDeploymentInitContainers = "/spec/template/spec/initContainers"
	pathPodInitContainers        = "/spec/initContainers"
)

func init() {
	runtimeScheme := runtime.NewScheme()
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)

	deserializer = serializer.NewCodecFactory(runtimeScheme).UniversalDeserializer()
}

func getAnnotationValue(ignoredNamespaceList []string, metadata *metav1.ObjectMeta, spec *corev1.PodSpec) (string, bool) {

	// check if InitContainer already injected
	for _, c := range spec.InitContainers {
		if c.Name == initContainerName {
			return "", false
		}
	}

	// skip special kubernetes system namespaces
	for _, namespace := range ignoredNamespaceList {
		if metadata.Namespace == namespace {
			logrus.Infof("Skip validation for %v for it's in special namespace:%v", metadata.Name, metadata.Namespace)
			return "", false
		}
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		return "", false
	}

	value, ok := annotations[nsmAnnotationKey]
	return value, ok
}

func validateAnnotationValue(value string) error {
	urls, err := tools.ParseAnnotationValue(value)
	logrus.Infof("Annotation nsurls: %v", urls)
	return err
}

func createPatch(annotationValue string, path string) ([]byte, error) {
	var patch []patchOperation

	value := []interface{}{
		map[string]interface{}{
			"name":            initContainerName,
			"image":           fmt.Sprintf("%s/%s:%s", repo, initContainer, tag),
			"imagePullPolicy": "IfNotPresent",
			"env": []interface{}{
				map[string]string{
					"name":  client.AnnotationEnv,
					"value": annotationValue,
				},
			},
			"resources": map[string]interface{}{
				"limits": map[string]interface{}{
					"networkservicemesh.io/socket": 1,
				},
			},
		},
	}

	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  path,
		Value: value,
	})

	return json.Marshal(patch)
}
