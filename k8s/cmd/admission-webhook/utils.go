package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
)

type podSpecAndMeta struct {
	meta *metav1.ObjectMeta
	spec *corev1.PodSpec
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
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

func getInitContainerPatchPath(request *v1beta1.AdmissionRequest) string {
	return getSpecPath(request) + "/initContainers"
}

func getSpecPath(request *v1beta1.AdmissionRequest) string {
	if request.Kind.Kind == pod {
		return pathPodSpec
	}
	if request.Kind.Kind == deployment {
		return pathDeploymentSpec
	}

	panic("unsupported request kind")
}

func validateAnnotationValue(value string) error {
	urls, err := tools.ParseAnnotationValue(value)
	logrus.Infof("Annotation result: %v", urls)
	return err
}

func createInitContainerPatch(annotationValue, path string, secure bool) []patchOperation {
	var patch []patchOperation

	envVals := []corev1.EnvVar{{
		Name:  client.AnnotationEnv,
		Value: annotationValue,
	},
	}
	jaegerHost := getJaegerHost()
	if jaegerHost != "" {
		envVals = append(envVals,
			corev1.EnvVar{
				Name:  jaegerHostEnv,
				Value: jaegerHost,
			})
	}
	jaegerPort := getJaegerPort()
	if jaegerPort != "" {
		envVals = append(envVals,
			corev1.EnvVar{
				Name:  jaegerPortEnv,
				Value: jaegerPort,
			})
	}

	var volumeMounts []corev1.VolumeMount
	if secure {
		volumeMounts = []corev1.VolumeMount{
			{
				Name:      spireSocketVolume,
				MountPath: spireSocketPath,
				ReadOnly:  true,
			},
		}
	}

	value := []corev1.Container{{
		Name:            initContainerName,
		Image:           fmt.Sprintf("%s/%s:%s", getRepo(), getInitContainer(), getTag()),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env:             envVals,
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"networkservicemesh.io/socket": resource.MustParse("1"),
			},
		},
		VolumeMounts: volumeMounts,
	}}

	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  path,
		Value: value,
	})

	return patch
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

func isIgnoreNamespace(ignoredNamespaceList []string, tuple *podSpecAndMeta) bool {
	for _, namespace := range ignoredNamespaceList {
		if tuple.meta.Namespace == namespace {
			return true
		}
	}
	return false
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

func addVolumeMounts(spec *corev1.PodSpec, added corev1.VolumeMount, basePath string) (patch []patchOperation) {
	for i := 0; i < len(spec.Containers); i++ {
		path := basePath + "/containers/" + strconv.Itoa(i) + "/volumeMounts"
		patch = append(patch, addVolumeMount(added, path, len(spec.Containers[i].VolumeMounts) == 0)...)
	}

	for i := 0; i < len(spec.InitContainers); i++ {
		path := basePath + "/initContainers/" + strconv.Itoa(i) + "/volumeMounts"
		patch = append(patch, addVolumeMount(added, path, len(spec.InitContainers[i].VolumeMounts) == 0)...)
	}

	return
}

func addVolumeMount(added corev1.VolumeMount, path string, first bool) (patch []patchOperation) {
	var value interface{}

	if first {
		value = []corev1.VolumeMount{added}
	} else {
		path = path + "/-"
		value = added
	}

	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  path,
		Value: value,
	})

	return
}

func addVolume(spec *corev1.PodSpec, added corev1.Volume, basePath string) (patch []patchOperation) {
	path := basePath + "/volumes"
	var value interface{}

	if len(spec.Volumes) == 0 {
		value = []corev1.Volume{added}
	} else {
		path = path + "/-"
		value = added
	}

	patch = append(patch, patchOperation{
		Op:    "add",
		Path:  path,
		Value: value,
	})

	return
}
