# Quick Start Network Service Mesh

This document will configure two Vagrant boxes in a Kubernetes cluster with a master and a worker node. It will also deploy the Network Service Mesh components in the cluster.

### Pre-requisites

* [Vagrant](https://www.vagrantup.com/docs/installation/)
* [Docker](https://docs.docker.com/install/)

```bash
git clone https://github.com/ligato/networkservicemesh && cd networkservicemesh
```

### Build

First let's build the Docker images of the various components

```bash
make k8s-save
```

### Deploy

Then we'll configure a Kubernetes cluster. A master and a worker node will be launched in two separate Vagrant machines. The Network Service Mesh components will then be deployed to the cluster.

```bash
make k8s-deploy
```

### Verify the daemonset is running

You can configure your Kubernetes client to interact with the cluster directly.

```bash
source $GOPATH/src/github.com/ligato/networkservicemesh/scripts/vagrant/env.sh
kubectl get daemonset nsmd
```

 This will allow you to see your Network Service Mesh daemonset running:

```bash
NAME   DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
nsmd   2         2         2       2            2           <none>          19m
```

### Test

To verify the setup is working, let's send some pings between the two nodes.

```bash
make k8s-check
```

You should see ping succeeding as below:

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