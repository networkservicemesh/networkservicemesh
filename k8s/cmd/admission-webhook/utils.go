package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/networkservicemesh/networkservicemesh/k8s/pkg/networkservice/namespace"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
)

type podSpecAndMeta struct {
	meta *metav1.ObjectMeta
	spec *corev1.PodSpec
}

func applyDeploymentKind(patches []patchOperation, kind string) {
	if kind == pod {
		return
	}
	if kind != deployment {
		logrus.Fatalf(unsupportedKind, kind)
	}
	for i := 0; i < len(patches); i++ {
		patches[i].Path = deploymentSubPath + patches[i].Path
	}
}

func defaultDNSSearchDomains() []string {
	return []string{fmt.Sprintf("%v.svc.cluster.local", namespace.GetNamespace()), "svc.cluster.local", "cluster.local"}
}

func getMetaAndSpec(request *v1beta1.AdmissionRequest) (*podSpecAndMeta, error) {
	result := &podSpecAndMeta{}
	if request.Kind.Kind == deployment {
		var deployment appsv1.Deployment
		if err := json.Unmarshal(request.Object.Raw, &deployment); err != nil {
			logrus.Errorf("Could not unmarshal raw object: %v", err)
			return nil, err
		}
		result.meta = &deployment.ObjectMeta
		result.spec = &deployment.Spec.Template.Spec
	}
	if request.Kind.Kind == pod {
		var pod corev1.Pod
		if err := json.Unmarshal(request.Object.Raw, &pod); err != nil {
			logrus.Errorf("Could not unmarshal raw object: %v", err)
			return nil, err
		}
		result.meta = &pod.ObjectMeta
		result.spec = &pod.Spec
	}
	return result, nil
}
func validateAnnotationValue(value string) error {
	urls, err := tools.ParseAnnotationValue(value)
	logrus.Infof("Annotation result: %v", urls)
	return err
}

func checkNsmInitContainerDuplication(spec *corev1.PodSpec) error {
	for i := 0; i < len(spec.InitContainers); i++ {
		c := &spec.InitContainers[i]
		if c.Name == getInitContainer() {
			return errors.New("do not use init-container and nsm annotation\nplease remove annotation or init-container")
		}
	}
	return nil
}

func isSupportKind(request *v1beta1.AdmissionRequest) bool {
	return request.Kind.Kind == pod || request.Kind.Kind == deployment
}

func getNsmAnnotationValue(ignoredNamespaceList []string, tuple *podSpecAndMeta) (string, bool) {
	// skip special kubernetes system namespaces
	for _, ignoredNamespace := range ignoredNamespaceList {
		if tuple.meta.Namespace == ignoredNamespace {
			logrus.Infof("Skip validation for %v for it's in special namespace:%v", tuple.meta.Name, tuple.meta.Namespace)
			return "", false
		}
	}

	annotations := tuple.meta.GetAnnotations()
	if annotations == nil {
		logrus.Info("No annotations, skip")
		return "", false
	}

	value, ok := annotations[nsmAnnotationKey]
	return value, ok
}

func parseAdmissionReview(body []byte) (*v1beta1.AdmissionReview, error) {
	r := &v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, r); err != nil {
		return nil, err
	}
	return r, nil
}

func readRequest(r *http.Request) ([]byte, error) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		logrus.Error(emptyBody)
		return nil, errors.New(emptyBody)
	}
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		msg := fmt.Sprintf(invalidContentType, contentType)
		logrus.Error(msg)
		return nil, errors.New(msg)
	}
	return body, nil
}
