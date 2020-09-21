package main

import v1 "k8s.io/api/core/v1"

const (
	emptyBody               = "empty body"
	mutateMethod            = "/mutate"
	invalidContentType      = "invalid Content-Type=%v, expect \"application/json\""
	couldNotEncodeReview    = "could not encode response: %v"
	couldNotWriteReview     = "could not write response: %v"
	deployment              = "Deployment"
	pod                     = "Pod"
	nsmAnnotationKey        = "ns.networkservicemesh.io"
	repoEnv                 = "REPO"
	initContainerEnv        = "INITCONTAINER"
	namespaceEnv            = "NSM_NAMESPACE"
	tagEnv                  = "TAG"
	tracerEnabledEnv        = "TRACER_ENABLED"
	jaegerHostEnv           = "JAEGER_AGENT_HOST"
	jaegerPortEnv           = "JAEGER_AGENT_PORT"
	enforceLimitsEnv        = "ENFORCE_LIMITS"
	repoDefault             = "networkservicemesh"
	initContainerDefault    = "nsm-init"
	dnsInitContainerDefault = "nsm-dns-init"
	namespaceDefault        = "default"
	tagDefault              = "latest"
	initContainerName       = "nsm-init-container"
	certFile                = "/etc/webhook/certs/" + v1.TLSCertKey
	keyFile                 = "/etc/webhook/certs/" + v1.TLSPrivateKeyKey
	initContainersPath      = "/spec/initContainers"
	unsupportedKind         = "kind %v is not supported"
	deploymentSubPath       = "/spec/template"
	volumePath              = "/spec/volumes"
	containersPath          = "/spec/containers"
	defaultPort             = 443

	// Keep in sync with ../../../test/kubetest/pods/limits.go.
	// Limits for 'nsm-monitor' container.
	nsmMonitorCPULimit    = "100m"
	nsmMonitorMemoryLimit = "15Mi"
	// Limits for 'coredns' container.
	corednsCPULimit    = "100m"
	corednsMemoryLimit = "15Mi"
	// Limits for 'nsm-init' container.
	nsmInitCPULimit    = "500m"
	nsmInitMemoryLimit = "20Mi"
	// Limits for 'nsm-dns-init' container.
	nsmDNSInitCPULimit    = "500m"
	nsmDNSInitMemoryLimit = "15Mi"
)
