package main

import (
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	v1 "k8s.io/kubernetes/pkg/apis/core/v1"
)

var (
	deserializer      runtime.Decoder
	ignoredNamespaces = []string{
		metav1.NamespaceSystem,
		metav1.NamespacePublic,
	}
)

func init() {
	runtimeScheme := runtime.NewScheme()
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// TODO: remove workaround if issue will be fixed
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)

	deserializer = serializer.NewCodecFactory(runtimeScheme).UniversalDeserializer()
}
