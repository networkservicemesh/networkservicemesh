package main

import v1 "k8s.io/api/core/v1"

const (
	emptyBody            = "empty body"
	mutateMethod         = "/mutate"
	invalidContentType   = "invalid Content-Type=%v, expect \"application/json\""
	couldNotEncodeReview = "could not encode response: %v"
	couldNotWriteReview  = "could not write response: %v"
	deployment           = "Deployment"
	pod                  = "Pod"
	nsmAnnotationKey     = "ns.networkservicemesh.io"
	repoEnv              = "REPO"
	initContainerEnv     = "INITCONTAINER"
	namespaceEnv         = "NSM_NAMESPACE"
	tagEnv               = "TAG"
	tracerEnabledEnv     = "TRACER_ENABLED"
	jaegerHostEnv        = "JAEGER_AGENT_HOST"
	jaegerPortEnv        = "JAEGER_AGENT_PORT"
	repoDefault          = "networkservicemesh"
	initContainerDefault = "nsm-init"
	namespaceDefault     = "default"
	tagDefault           = "latest"
	initContainerName    = "nsm-init-container"
	dnsInitContainerName = "nsm-dns-init"
	certFile             = "/etc/webhook/certs/" + v1.TLSCertKey
	keyFile              = "/etc/webhook/certs/" + v1.TLSPrivateKeyKey
	initContainersPath   = "/spec/initContainers"
	unsupportedKind      = "kind %v is not supported"
	deploymentSubPath    = "/spec/template"
	volumePath           = "/spec/volumes"
	containersPath       = "/spec/containers"
	defaultPort          = 443
)
