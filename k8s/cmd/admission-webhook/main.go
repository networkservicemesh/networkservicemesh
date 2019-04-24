package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/client"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	v1 "k8s.io/kubernetes/pkg/apis/core/v1"
	"net/http"
	"os"
)

const (
	certFile = "/etc/webhook/certs/cert.pem"
	keyFile  = "/etc/webhook/certs/key.pem"
)

type WebhookServer struct {
	server *http.Server
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

var (
	ignoredNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}
)

var (
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

	pathDeploymentInitContainers = "/spec/template/spec/initContainers"
	pathPodInitContainers        = "/spec/initContainers"
)

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)
}

func getAnnotationValue(ignoredList []string, metadata *metav1.ObjectMeta, spec *corev1.PodSpec) (string, bool) {

	// check if InitContainer already injected
	for _, c := range spec.InitContainers {
		if c.Name == "nsc" {
			return "", false
		}
	}

	// skip special kubernetes system namespaces
	for _, namespace := range ignoredList {
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
			"name":            "nsc",
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

func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request

	logrus.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v UID=%v patchOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, req.UID, req.Operation, req.UserInfo)

	var meta *metav1.ObjectMeta
	var spec *corev1.PodSpec
	var path string

	switch req.Kind.Kind {
	case "Deployment":
		var deployment appsv1.Deployment
		if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
			logrus.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		meta = &deployment.ObjectMeta
		spec = &deployment.Spec.Template.Spec
		path = pathDeploymentInitContainers
	case "Pod":
		var pod corev1.Pod
		if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
			logrus.Errorf("Could not unmarshal raw object: %v", err)
			return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
		meta = &pod.ObjectMeta
		spec = &pod.Spec
		path = pathPodInitContainers
	default:
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	value, ok := getAnnotationValue(ignoredNamespaces, meta, spec)

	if !ok {
		logrus.Infof("Skipping validation for %s/%s due to policy check", meta.Namespace, meta.Name)
		return &v1beta1.AdmissionResponse{
			Allowed: true,
		}
	}

	err := validateAnnotationValue(value)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	patchBytes, err := createPatch(value, path)
	if err != nil {
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	logrus.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed: true,
		Patch:   patchBytes,
		PatchType: func() *v1beta1.PatchType {
			pt := v1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func (whsvr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		logrus.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		logrus.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		logrus.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = whsvr.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		logrus.Errorf("Can't encode response: %v", err)
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	if _, err := w.Write(resp); err != nil {
		logrus.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func main() {
	// Capture signals to cleanup before exiting
	c := tools.NewOSSignalChannel()

	logrus.Info("Admission Webhook starting...")

	repo = os.Getenv(repoEnv)
	if repo == "" {
		repo = repoDefault
	}

	initContainer = os.Getenv(initContainerEnv)
	if initContainer == "" {
		initContainer = initContainerDefault
	}

	tag = os.Getenv(tagEnv)
	if tag == "" {
		tag = tagDefault
	}

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		logrus.Fatalf("Failed to load key pair: %v", err)
	}

	whsvr := &WebhookServer{
		server: &http.Server{
			Addr:      fmt.Sprintf(":%v", 443),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.serve)
	whsvr.server.Handler = mux

	// start webhook server in new routine
	go func() {
		if err := whsvr.server.ListenAndServeTLS("", ""); err != nil {
			logrus.Fatalf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	logrus.Info("Server started")
	<- c
}
