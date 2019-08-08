package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/go-errors/errors"
	"k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type podSpecAndMeta struct {
	meta *metav1.ObjectMeta
	spec *corev1.PodSpec
	name string
}
type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func parseAdmissionReview(body []byte) (*v1beta1.AdmissionReview, error) {
	r := &v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, r); err != nil {
		return nil, err
	}
	return r, nil
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
		result.name = pod.Name
		result.meta = &pod.ObjectMeta
		result.spec = &pod.Spec
	}
	return result, nil
}
func readRequest(r *http.Request) ([]byte, error) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		return nil, errors.New("empty body")
	}
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return nil, errors.New(fmt.Sprintf("invalid content type %v", contentType))
	}
	return body, nil
}

func addVolumeMounts(spec *corev1.PodSpec, added []corev1.VolumeMount, init bool) (patch []patchOperation) {
	for i := 0; i < len(spec.Containers); i++ {
		container := &spec.Containers[i]
		path := containersPath + "/" + strconv.Itoa(i) + "/volumeMounts"
		if init {
			path = intContainersPath + "/" + strconv.Itoa(i) + "/volumeMounts"
		}
		first := len(container.VolumeMounts) == 0
		if !first {
			path = path + "/-"
		}
		for _, v := range added {
			patch = append(patch, patchOperation{
				Op:    "add",
				Path:  path,
				Value: v,
			})
		}
	}
	return patch
}

func addEnv(spec *corev1.PodSpec, added []corev1.EnvVar, init bool) (patch []patchOperation) {
	for i := 0; i < len(spec.Containers); i++ {
		container := &spec.Containers[i]
		path := containersPath + "/" + strconv.Itoa(i) + "/env"
		if init {
			path = intContainersPath + "/" + strconv.Itoa(i) + "/env"
		}
		first := len(container.VolumeMounts) == 0
		if !first {
			path = path + "/-"
		}
		for _, v := range added {
			patch = append(patch, patchOperation{
				Op:    "add",
				Path:  path,
				Value: v,
			})
		}
	}
	return patch
}

func addVolume(spec *corev1.PodSpec, added []corev1.Volume) (patch []patchOperation) {
	first := len(spec.Volumes) == 0
	var value interface{}
	for i := 0; i < len(added); i++ {
		value = added[i]
		path := volumePath
		if first {
			first = false
			value = []corev1.Volume{added[i]}
		} else {
			path = path + "/-"
		}
		patch = append(patch, patchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		})
	}
	return patch
}

func applyDeploymentKind(patches []patchOperation, kind string) {
	if kind == pod {
		return
	}
	if kind != deployment {
		panic(fmt.Sprintf("not supproted %v", kind))
	}
	for i := 0; i < len(patches); i++ {
		patches[i].Path = deploymentSubPath + patches[i].Path
	}
}
