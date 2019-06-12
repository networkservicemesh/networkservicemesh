# Configure Network Service Mesh with a Kernel-based forwarding plane

The default forwarding plane in Network Service Mesh is VPP.

## Kernel-based forwarding plane

Another alternative is a Kernel-based forwarding plane.

Disclaimer:
Note that it is still under development.
For the time being, it supports only local connections.

## How to configure

To configure Network Service Mesh with it, you can use the following environment variables.

For example:

```bash
WORKER_COUNT=0 FORWARDING_PLANE=kernel-forwarder make k8s-save
```

* WORKER_COUNT stands for the number of worker hosts that we have.
* FORWARDING_PLANE stands for the forwarding plane that we want to use. 

## Give it a try

To try it yourself, you can clone the project and do the following:

Create your Kubernetes cluster using Vagrant
```bash
WORKER_COUNT=0 FORWARDING_PLANE=kernel-forwarder make vagrant-start
```
build and save all the necessary images
```bash
WORKER_COUNT=0 FORWARDING_PLANE=kernel-forwarder make k8s-save
```
load the images inside the VM nodes
```bash
WORKER_COUNT=0 FORWARDING_PLANE=kernel-forwarder make k8s-load-images
```
deploy Network Service Mesh
```bash
WORKER_COUNT=0 FORWARDING_PLANE=kernel-forwarder make k8s-infra-deploy
```
deploy the ICMP example
```bash
WORKER_COUNT=0 FORWARDING_PLANE=kernel-forwarder make k8s-icmp-deploy
```
and finally verify that it all works as expected:
```bash
WORKER_COUNT=0 FORWARDING_PLANE=kernel-forwarder make k8s-check
```