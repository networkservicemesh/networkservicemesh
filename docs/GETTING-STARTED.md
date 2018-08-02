# Getting started with Network Service Mesh

This document illustrates the procedure to install and start network service mesh from a bare bones system.

## Table of Contents
1. [Introduction](#"Introduction")
2. [Quick Start](#'quick')
3. [Set up Virtualization](#"Nested')
4. [Install on Ubuntu](#"Ubuntu")
5. [Install on Fedora.](#"Fedora")
6. [Install pre-requisites](#"prereq")
7. [Install docker](#"docker")
8. [Install docker CE](#"dockerCE")
9. [Install kubernetes](#'kube')
10. [Install virt and kvm](#'virt')
11. [Install kubectl](#'kubectl')
12. [Install golang](#'golang')
13. [Install minikube ](#'minikube')
14. [Install kvm2 driver](#'kvm2')
15. [Local docker images](#'Local')
16. [Install protobuf](#'proto')
17. [Install NSM](#'NSM')
18. [Verify NSM](#'NSMverify')
19. [Simple Make](#'simplemake')
20. [Run NSM](#'runNSM')
21. [Manual steps](#'Manual')
22. [TODO](#'TODO')
23. [Credits](#'credit')

## Introduction <a name="Introduction"></a>

For a working example we will launch a VM running kubernetes, and all required packages as well as network service mesh itself. The exampe starts with launching a VM under Qemu/KVM. You may of course choose a different hypervisor but those procedures haven't been verified in this document.

You may use your favorite distro. In this document we hope to show the procedures for Ubuntu, Centos, Fedora, RHEL, and perhaps others.

The steps here have analogies for other distros.  Also, this procedure uses the minikube cluster with the kvm2 docker driver. This may provide a more interesting case and probably is a closer simulation of what may happen on a "real"
 scaled cloud, private or public.

## Quick Start

If you may prefer the fast path straight to running. Assuming you already have a kubernetes cluster, probably minikube installed  and all packages such as golang, docker and kubernetes installed. Once you have installed network service mesh source code, go to straght to [Quick Start](#'quick')

## Set up Nested Virtualization <a name="Nested"></a>

For an example, install in a x86_64 VM running under Qemu/KVM.

 If you choose to install on a VM, be advised that we are using minikube. minikube creates its own VM to run the cluster. You must make sure you are set up for nested virtualization.
 If you are using virtlib on x86_64, edit the xml to add vmx to cpu features.

```
  <cpu mode='custom' match='exact' check='full'>
    <model fallback='forbid'>Broadwell</model>
    <feature policy='require' name='vmx'/>
    ...
```

### Install on deb based distro <a name="deb"></a>

#### Install on Ubuntu <a name="Ubuntu"></a>

TBD

### Install on RPM based distros <a name="RPM"></a>

#### Install on Centos or RHEL <a name="Centos"></a>

TBD

#### Install on Fedora <a name="Fedora"></a>

#### Install pre-requisites <a name="prereq"></a>

In this example we used Fedora 28.

You will need sudo privileges.

```
sudo dnf update -y
sudo dnf -y groupinstall 'C Development Tools and Libraries'
sudo dnf install -y yum-utils
sudo dnf install -y git
sudo dnf install -y ShellCheck pv
```

Set selinux to Permissive.

```
sudo setenforce 0
```

### Install docker <a name="docker"></a>

Install version from Fedora repo or install from Docker CE upstream.

#### Install current version from Fedora repo.

```
sudo dnf install -y docker
```

#### Or install docker CE <a name="dockerCE"></a>

The current scripts and dockerfiles don't work to well with the older version of docker from the Fedora 28 repo.

As an alternative, install from Docker CE upstream. First remove any old versions of docker.

````
sudo dnf remove -y docker \
                  docker-client \
                  docker-client-latest \
                  docker-common \
                  docker-latest \
                  docker-latest-logrotate \
                  docker-logrotate \
                  docker-selinux \
                  docker-engine-selinux \
                  docker-engine

sudo dnf config-manager \
    --add-repo \
    https://download.docker.com/linux/fedora/docker-ce.repo
sudo dnf config-manager --set-enabled docker-ce-edge
sudo dnf install -y docker-ce
````

Now, add yourself to a docker group so you can run docker commands without sudo.

```
sudo groupadd docker
sudo usermod -a -G docker $(whoami)
```

Logout and back in again. Also restart docker.

```
sudo systemctl restart docker
```

Test docker

```
docker run hello-world
```

#### Install kubernetes as documented in kubernetes.io <a name="kube"></a>

The following procedure installs kubernetes as a single-node cluster
[https://kubernetes.io/docs/setup/pick-right-solution/#local-machine-solutions]


First install virt and kubectl. Then install golang and minikube.

### Install virt and kvm <a name="virt"></a>

```
sudo dnf install -y qemu-kvm qemu-img libvirt libvirt-python libvirt-client virt-install virt-viewer bridge-utils
sudo systemctl start libvirtd
sudo systemctl enable libvirtd
```

[Documented in more detail here:](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

#### Install kubectl <a name="kubectl"></a>

Install upstream kubernetes.io repo for the most recent packages.

```
sudo vi /etc/yum.repos.d/kubernetes.repo
[kubernetes]
name=Kubernetes
baseurl=https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=1
gpgkey=https://packages.cloud.google.com/yum/doc/yum-key.gpg https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
```

```
sudo dnf config-manager --set-enabled kubernetes
sudo dnf install -y kubectl
```

### Install golang <a name="golang"></a>

[Using these instructions from the go project.](https://go-repo.io/)
However, we will use the version of golang in the Fedora distro.

```
sudo dnf install -y golang
    
mkdir -p ~/go/{bin,pkg,src}
echo 'export GOPATH="$HOME/go"' >> ~/.bashrc
echo 'export PATH="$PATH:${GOPATH//://bin:}/bin"' >> ~/.bashrc
source $HOME/.bashrc
```

To test go installation:

```
mkdir -p ~/go/src/hello
vi ~/go/src/hello/hello.go

    package main
    
    import "fmt"
    
    func main() {
            fmt.Printf("hello, world\n")
    }


cd ~/go/src/hello
go build
./hello
```

### Install minikube <a name="minikube"></a>

Install minikube with the Kubernetes kvm2 driver for VM Support. Install using kvm2 driver as recommended by kubernetes upstream documentation.

[https://github.com/kubernetes/minikube/blob/v0.28.0/README.md]

There is also a copr to install on Fedora [here.](https://copr.fedorainfracloud.org/coprs/antonpatsev/minikube-rpm/)

```
curl -Lo minikube https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64 && chmod +x minikube && sudo mv minikube /usr/local/bin/

echo 'export MINIKUBE_WANTUPDATENOTIFICATION=false' >> ~/.bashrc
echo 'export MINIKUBE_WANTREPORTERRORPROMPT=false' >> ~/.bashrc
echo 'export MINIKUBE_HOME=$HOME' >> ~/.bashrc
echo 'export CHANGE_MINIKUBE_NONE_USER=true' >> ~/.bashrc
mkdir -p $HOME/.kube
touch $HOME/.kube/config
echo 'export KUBECONFIG=$HOME/.kube/config' >> ~/.bashrc
source ~/.bashrc
```

Now, add yourself or the user running kubectl to a libvirt group so you can run the libvirt commands without sudo.

```
sudo usermod -a -G libvirt $(whoami)
newgrp libvirt
```

#### Install kvm2 driver <a name="kvm2"></a>

```
curl -LO https://storage.googleapis.com/minikube/releases/latest/docker-machine-driver-kvm2 && chmod +x docker-machine-driver-kvm2 && sudo mv docker-machine-driver-kvm2 /usr/local/bin/
```
     
To check your GID membership for libvirt and docker

```
id $(whoami)
```

Start minikube

After step above, minikube which was downloaded previously should be in /usr/local/bin.

```
minikube start --vm-driver kvm2
```

You should see:

```
    Starting local Kubernetes v1.10.0 cluster...
    Starting VM...
        Minikube ISO
          153.08 MB / 153.08 MB [============================================] 100.00% 0s

    Getting VM IP address...
    Moving files into cluster...
    Downloading kubeadm v1.10.0
    Downloading kubelet v1.10.0
    Finished Downloading kubelet v1.10.0
    Finished Downloading kubeadm v1.10.0
    Setting up certs...
    Connecting to cluster...
    Setting up kubeconfig...
    Starting cluster components...
    Kubectl is now configured to use the cluster.
    Loading cached images from config file.
```

At any time you can verify that the minikube cluster is running by checking for the VM.

```
sudo virsh list --all
```

You should see:

```
    Id    Name                           State
    ----------------------------------------------------
    1     minikube                       running
```
#### Local docker images <a name="Local"></a>

If have local docker images that havn't been pushed into the docker repo, minikube will be able to "see" yourlocally built docker images if you set the docker environment variables for minikube kubernetes cluster.
```
eval $(minikube docker-env)
```
### Install protobuf <a name="proto"></a>

Install Google protobuf packages.

```
sudo dnf install -y protobuf dep
sudo dnf install -y protoc-gen-go
sudo dnf install -y protobuf-compiler
```

## Install and test network service mesh <a name="NSM"></a>

### Download Network Service Mesh Code

if GOPATH is set as environment variable

```
cd $GOPATH/src
go get github.com/ligato/networkservicemesh
```

### Get the code for Network Service Mesh!

Use make from the top level to generate the code, build docker images and install into minikube.

```
cd $GOPATH/src/github.com/ligato/networkservicemesh
```

#### Verify the code <a name="NSMverify"></a>

To verify the code

```
make verify

```

#### Build the docker images

```
make docker-build
```

### Combine above steps. <a name="simplemake"></a>

The default make target will execute all the above steps and verify the code and build the docker images.

```
make
```
### Run network service mesh <a name="runNSM"></a>

This step will actually start the Network Service Mesh daemonset into the minikube cluster.
First, make sure to label the nodes where you want to run the image:

```
kubectl label --overwrite nodes minikube app=networkservice-node
```
#### Now, create the daemonset <a name="quick"></a>

If you already have minikube, docker, golang and the network service mesh code installed, you may start
NSM in one command. If not, review the steps above.

```
kubectl create -f conf/sample/networkservice-daemonset.yaml

```

Now you should be able to see your Network Service Mesh daemonset running:

```
cd $GOPATH/github.com/ligato/networkservicemesh
kubectl get pods
```
You should see:
```
NAME                   READY     STATUS    RESTARTS   AGE
networkservice-x5k9s   1/1       Running   0          5h
```


## "Manual" execution using scripts and lower level commands <a name="Manual"></a>

You may also several of the above steps manually.

#### Update after Kubernetes refresh

You may need to generate generate code after a Kubernetes refresh The procedure below will
generate deep copy functions, clientset, listeners and informers as follows.

```
cd $GOPATH/src/github.com/ligato/networkservicemesh

go generate ./...
./scripts/update-codegen.sh
go build ./...
go test ./...
```

You should see something like:

```
?   	github.com/ligato/networkservicemesh/cmd/netmesh	[no test files]
?   	github.com/ligato/networkservicemesh/cmd/nse	[no test files]
ok  	github.com/ligato/networkservicemesh/cmd/nsm-init	0.021s
?   	github.com/ligato/networkservicemesh/pkg/apis/networkservicemesh.io/v1	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/clientset/versioned	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/clientset/versioned/fake	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/clientset/versioned/scheme	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/clientset/versioned/typed/networkservicemesh.io/v1	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/clientset/versioned/typed/networkservicemesh.io/v1/fake	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/informers/externalversions	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/informers/externalversions/internalinterfaces	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/informers/externalversions/networkservicemesh.io	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/informers/externalversions/networkservicemesh.io/v1	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/client/listers/networkservicemesh.io/v1	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/nsm/apis/common	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/nsm/apis/netmesh	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/nsm/apis/nseconnect	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/nsm/apis/nsmconnect	[no test files]
?   	github.com/ligato/networkservicemesh/pkg/nsm/apis/pod2nsm	[no test files]
ok  	github.com/ligato/networkservicemesh/plugins/config	0.019s
ok  	github.com/ligato/networkservicemesh/plugins/crd	0.041s
ok  	github.com/ligato/networkservicemesh/plugins/handler	0.038s
?   	github.com/ligato/networkservicemesh/plugins/idempotent	[no test files]
ok  	github.com/ligato/networkservicemesh/plugins/interupthandler	0.041s
ok  	github.com/ligato/networkservicemesh/plugins/logger	0.028s
?   	github.com/ligato/networkservicemesh/plugins/logger/hooks/pid	[no test files]
ok  	github.com/ligato/networkservicemesh/plugins/nsmcommand	0.024s
ok  	github.com/ligato/networkservicemesh/plugins/nsmserver	0.010s
ok  	github.com/ligato/networkservicemesh/plugins/objectstore	0.006s
?   	github.com/ligato/networkservicemesh/utils/command	[no test files]
ok  	github.com/ligato/networkservicemesh/utils/idempotent	0.028s
```

Regenerate the deepcopy code needed for network service mesh CRD code.

```
$GOPATH/bin/deepcopy-gen --input-dirs ./netmesh/model/netmesh --go-header-file conf/boilerplate.txt --bounding-dirs ./netmesh/model/netmesh -O zz_generated.deepcopy -o $GOPATH/src
```

Verify the generated code manually

```
./scripts/verify-codegen.sh
```

#### Build the docker images manually

```
cd $GOPATH/src/github.com/ligato/networkservicemesh
docker build -t ligato/networkservicemesh/netmesh-test -f build/nsm/docker/Test.Dockerfile .
```

## TODO <a name="TODO"></a>

* Add procedure for Ubuntu.
* Add procedure for Centos.
* Add procedure for RHEL.
* List all package versions.

#### Credits <a name="credit"></a>
This stuff was largely cribbed from other work and to reviewers, testers and deployers. Also, thanks to the NSM community and to previous writers of the build, readme and run docs. Also, this was gleaned from upstream documentation in golang kubernetes and docker.
