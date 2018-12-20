# Quick Start Network Service Mesh

This document will configure two Vagrant boxes in a Kubernetes cluster with a master and a worker node. It will also deploy the Network Service Mesh components in the cluster.

The following instructions assume Ubuntu 18.04.

### Ubuntu pre-requisites installation

To get the latest Vagrant verision we propose adding the following repo:

```bash
sudo bash -c 'echo deb https://vagrant-deb.linestarve.com/ any main > /etc/apt/sources.list.d/wolfgang42-vagrant.list'
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-key AD319E0F7CFFA38B4D9F6E55CE3F3DE92099F7A4
```

The latest `kubectl` is available with this repo:

```bash
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add
sudo apt-add-repository "deb http://apt.kubernetes.io/ kubernetes-xenial main"
```

After adding these repos, one needs to update and install the required packages as follows:

```bash
sudo apt update
sudo apt install -y virtualbox vagrant docker.io kubectl
```

The Docker service needs to be enabled:

```bash
sudo systemctl enable docker
```

And then the current user should be added to the proper user group:

```bash
sudo usermod -aG docker $USER
```

Log out and log in again, so that the user group addition takes effect.

### Generic pre-requisites installation instructions

If you are not using Ubuntu, you can try following the generic installation instructions from here:

* [VirtualBox](https://www.virtualbox.org/wiki/Downloads)
* [Vagrant](https://www.vagrantup.com/docs/installation/)
* [Docker](https://docs.docker.com/install/)

### Getting the Network Service Mesh project

```bash
git clone https://github.com/ligato/networkservicemesh 
cd networkservicemesh
```

### Build

First let's build the Docker images of the various components

```bash
make k8s-save
```

### Local Vagrant setuo

Then we'll configure a Kubernetes cluster. A master and a worker node will be launched in two separate Vagrant machines. The Network Service Mesh components will then be deployed to the cluster.

```bash
make vagrant-start
```

Ensure your `kubectl` is configured to point to Vagrant's Kubernetes cluster.

```bash
source scripts/vagrant/env.sh
```

### Deploy the core Network Service Mesh components

The core Network Service Mesh infrastructure is deployed with the following command.

```bash
make k8s-infra-deploy
```

### Verify the services are up and running

A simple check should show two `nsmd`, two `nsm-vppagent-dataplane`, two `skydive-agent`, one `crossconnect-monitor` and one `skydive-analyzer` pods.

```bash
kubect get pods
```

This will allow you to see your Network Service Mesh daemonset running:

```bash
kubectl get daemonset nsmd


NAME   DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
nsmd   2         2         2       2            2           <none>          19m
```

### Running the Networking Service Mesh examples

The basic ICMP example is deployed like this:

```bash
make k8s-icmp-deploy
```

The VPN service composition example is deployed with:

```bash
make k8s-vpn-deploy
```

### Test

Both of the examples can be verified with running a simple check:

```bash
make k8s-check
```

For ICMP you should see ping succeeding as below:

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

Validating the VPN example will output the following:

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
