# Adding sidecar containers to application containers using NetworkServiceMesh(NSM)

In the current NSM architecture, any pod deployed on a node communicates with NSM or exposes a channel. Going forward there will be a lot functionality like GRPC message handling, exposing channels, sending requests to NSM on same node, which should be common to all application pods running in the cluster. To add this functionality to any pod without any code change, the easiest way is to add a sidecar container at the time a pod is being deployed by Kubernetes API server. This document captures the details for components and explains the functionality to add a sidecar container to a pod.

## Brief Overview of the approach

Starting from version 1.9 Kubernetes provides support (Beta) for MutatingAdmissionWebhook. This is added as a plugin to external-admission-webhooks supported by Kubernetes API server, and can be easily configured at the runtime without any recompilation for API server binary. The idea here is when a task is being created, such as pod creation a set of admission control plugins are being called, these plugins expose HTTP API which gives the response about the request received. If any one of the admission control plugin denies the creation, creation of the object fails. For a detailed explaination about this refer [kube-mutating-webhook-tutorial](https://github.com/morvencao/kube-mutating-webhook-tutorial) and [external-admission-webhooks](https://v1-8.docs.kubernetes.io/docs/admin/extensible-admission-controllers/#external-admission-webhooks)

## Components

* Sidecar container

  * Sidecar container image
  * Spec for container to be deployed

* MutatingWebhookConfiguration - for API server to know which endpoint to invoke.

* MutatingAdmissionWebhook Server

  * Pod running an HTTP server, exposing an API to inject sidecar
  * MutatingAdmissionWebhook service spec

* TLS for HTTP authentication between API server and MutatingAdmissionWebhook server

* Injector webhook configmap, which contains container spec information to be injected in target pod.


## How does it work

* MutatingAdmissionWebhook is registered with the API server and provides the API server with MutatingWebhookConfiguration. This configuration includes:
  
  * How does API server connects to MutatingAdmissionWebhook server
  * How to authenticate with the server
  * What actions/resources and selectors need to be forwarded to the MutatingAdmissionWebhook server

* When a pod gets created, admission control plugins which are configured in the API server are being called for mutation and validation.

* Once, this admission request is received by the MutatingAdmissionWebhook server, it can submit a PATCH to the admission request object adding new container spec to the API server.


## How does TLS authentication works

TODO