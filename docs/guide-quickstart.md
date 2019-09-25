# Network Service Mesh - Quick Start Guide

This document will help you configure two Vagrant boxes in a Kubernetes cluster with a master and a worker node. You will also deploy the Network Service Mesh components in the cluster.

## Prerequisites

You can find instructions for your operation systems in the links below:

* [CentOS](prereq-centos.md)
* [OSX](prereq-osx.md)
* [Ubuntu](prereq-ubuntu.md)

If you have another Linux distribution or prefer to go with the upstream, make sure you have the following dependencies installed:

* [VirtualBox](https://www.virtualbox.org/wiki/Downloads)
* [Vagrant](https://www.vagrantup.com/docs/installation/)
* [Docker](https://docs.docker.com/install/)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

### Getting the Network Service Mesh project

```bash
git clone https://github.com/networkservicemesh/networkservicemesh
cd networkservicemesh
```

## Build the Network Service Mesh images

First, let's build the Docker images of the various components:

```bash
make k8s-save
```

## Setup the local Vagrant environment

Then we'll configure a Kubernetes cluster. A master and a worker node will be launched in two separate Vagrant machines. The Network Service Mesh components will then be deployed to the cluster.

Start the Vagrant machines:

```bash
make vagrant-start
```

Ensure your `kubectl` is configured to point to Vagrant's Kubernetes cluster:

```bash
source scripts/vagrant/env.sh
```

## Deploy the core Network Service Mesh components

The core Network Service Mesh infrastructure is deployed with the following command:

```bash
make k8s-save
make k8s-load-images
make helm-init
make helm-install-nsm
```

### Verify the services are up and running

The following check should show two `nsmgr`, two `nsm-vpp-forwarder`, and one `nsm-admission-webhook` pod

```bash
kubectl get pods -n nsm-system

NAME                                     READY   STATUS    RESTARTS   AGE
nsm-admission-webhook-8597995474-2p7vc   1/1     Running   0          2m4s
nsm-vpp-forwarder-9lfnv                  1/1     Running   0          2m5s
nsm-vpp-forwarder-w424k                  1/1     Running   0          2m5s
nsmgr-5mkr2                              3/3     Running   0          2m5s
nsmgr-gc976                              3/3     Running   0          2m5s
```

This will allow you to see your Network Service Mesh daemonset running:

```bash
kubectl get daemonset nsmgr -n nsm-system


NAME   DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
nsmgr   2         2         2       2            2           <none>          19m
```

## Deploy the Monitoring components


```bash
helm-install-nsm-monitoring
```

When deployed successfully two `skydive-agent`, one `skydive-analyzer`, one `crossconnect-monitor` and one `jaeger` pod will be running in the nsm-system namespace.

````bash 
kubectl get pods -n nsm-system 

NAME                                     READY   STATUS    RESTARTS   AGE
crossconnect-monitor-57dcf588dd-qk2n9    1/1     Running   0          43s
jaeger-f5d6744c5-t2tc8                   1/1     Running   0          43s
skydive-agent-hr9xh                      1/1     Running   0          43s
skydive-agent-jxmm9                      1/1     Running   0          43s
skydive-analyzer-778fc98897-9cr5w        1/1     Running   0          43s
```

## Deploy the Network Service Mesh examples

Now that we have the NSM infrastructure deployed, we can proceed with deploying some of the examples.

* The basic ICMP example is deployed like this:

```bash
make helm-install-nsm
make helm-install-icmp-responder
```

* The VPN service composition example is deployed with:

```bash
make helm-install-nsm
make helm-install-vpn
```

## Verify deploying the examples

Both of the examples can be verified by running a simple check:

```bash
make k8s-check
```

* For ICMP you should see ping succeeding as below:

```bash
./scripts/nsc_ping_all.sh
===== >>>>> PROCESSING nsc-84799b768f-99wgf  <<<<< ===========
PING 10.20.1.2 (10.20.1.2): 56 data bytes
64 bytes from 10.20.1.2: seq=0 ttl=64 time=6.313 ms

--- 10.20.1.2 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max = 6.313/6.313/6.313 ms
NSC nsc-84799b768f-99wgf with IP 10.20.1.1/30 pinging icmp-responder-nse TargetIP: 10.20.1.2 successful
===== >>>>> PROCESSING nsc-84799b768f-d82w5  <<<<< ===========
PING 10.20.1.2 (10.20.1.2): 56 data bytes
64 bytes from 10.20.1.2: seq=0 ttl=64 time=1.015 ms

--- 10.20.1.2 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max = 1.015/1.015/1.015 ms
NSC nsc-84799b768f-d82w5 with IP 10.20.1.1/30 pinging icmp-responder-nse TargetIP: 10.20.1.2 successful
===== >>>>> PROCESSING nsc-84799b768f-ktm4w  <<<<< ===========
PING 10.30.1.2 (10.30.1.2): 56 data bytes
64 bytes from 10.30.1.2: seq=0 ttl=64 time=3.199 ms

--- 10.30.1.2 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max = 3.199/3.199/3.199 ms
NSC nsc-84799b768f-ktm4w with IP 10.30.1.1/30 pinging vppagent-icmp-responder-nse TargetIP: 10.30.1.2 successful
===== >>>>> PROCESSING nsc-84799b768f-wvxwt  <<<<< ===========
PING 10.30.1.2 (10.30.1.2): 56 data bytes
64 bytes from 10.30.1.2: seq=0 ttl=64 time=6.070 ms

--- 10.30.1.2 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max = 6.070/6.070/6.070 ms
NSC nsc-84799b768f-wvxwt with IP 10.30.1.1/30 pinging vppagent-icmp-responder-nse TargetIP: 10.30.1.2 successful
```

* Validating the VPN example will output the following:

```bash
===== >>>>> PROCESSING vpn-gateway-nsc-5458d48c86-zh4xf  <<<<< ===========
PING 10.60.1.2 (10.60.1.2): 56 data bytes
64 bytes from 10.60.1.2: seq=0 ttl=64 time=11.728 ms

--- 10.60.1.2 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max = 11.728/11.728/11.728 ms
NSC vpn-gateway-nsc-5458d48c86-zh4xf with IP 10.60.1.1/30 pinging vpn-gateway-nse TargetIP: 10.60.1.2 successful
Connecting to 10.60.1.2:80 (10.60.1.2:80)
null                 100% |*******************************|   112   0:00:00 ETA
NSC vpn-gateway-nsc-5458d48c86-zh4xf with IP 10.60.1.1/30 accessing vpn-gateway-nse TargetIP: 10.60.1.2 TargetPort:80 successful
Connecting to 10.60.1.2:8080 (10.60.1.2:8080)
wget: download timed out
command terminated with exit code 1
NSC vpn-gateway-nsc-5458d48c86-zh4xf with IP 10.60.1.1/30 blocked vpn-gateway-nse TargetIP: 10.60.1.2 TargetPort:8080
All check OK. NSC vpn-gateway-nsc-5458d48c86-zh4xf behaving as expected.
```
