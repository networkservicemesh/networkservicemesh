package main

const (
	certFile          = "/etc/webhook/certs/cert.pem"
	keyFile           = "/etc/webhook/certs/key.pem"
	deployment        = "Deployment"
	pod               = "Pod"
	volumePath        = "/spec/volumes"
	containersPath    = "/spec/containers"
	intContainersPath = "/spec/initContainers"
	deploymentSubPath = "/spec/template"
)
