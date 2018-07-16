# Install kubernetes on Centos/RHEL/Fedora.

## Before you begin

The following document shows installation on a bare bones VM running under Qemu/KVM. You may of course choose your distro and other hypervisor of choice.  The steps here have analogies for other distros.  Also, I used minicube cluster which provides a more interesting case and probably is a closer simulation of what may happen on a "real"
 scaled cloud, private or public.

For an example, install in a x86_64 Fedora 28 VM running under Qemu/KVM.

### Set up Nested Virtualization

 If you choose to install on a VM, be advised that we are using minikube. minikube creates a VM so make sure you are set up for nested virtualization.
 If you are using virtlib on x86_64, edit the xml to add vmx to cpu features.

```
  <cpu mode='custom' match='exact' check='full'>
    <model fallback='forbid'>Broadwell</model>
    <feature policy='require' name='vmx'/>
    ...
```
#### Install pre-requisites
You will need sudo priviliges.
```
sudo dnf update -y
sudo dnf -y groupinstall 'C Development Tools and Libraries'
sudo dnf install -y yum-utils
sudo dnf install -y git
sudo dnf install -y ShellCheck
```
Set selinux to Permissive.
```
sudo setenforce 0
```
#### Install docker

Install version from Fedora repo or install from Docker CE upstream.

Install current version from Fedora repo.

```
sudo dnf install -y docker
```
The current scripts and dockerfiles don't work to well with the older version of docker in the Fedora 28 repo.

As an alternative, install from Docker CE upstream. First remove any old versions of docker.
````
sudo dnf remove docker \
                  docker-client \
                  docker-client-latest \
                  docker-common \
                  docker-latest \
                  docker-latest-logrotate \
                  docker-logrotate \
                  docker-selinux \
                  docker-engine-selinux \
                  docker-engine
````
Install Docker CE
````
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

#### Install kubernetes as documented in kubernetes.io

The following procedure installs kubernetes as a single-node cluster
[https://kubernetes.io/docs/setup/pick-right-solution/#local-machine-solutions]


First install virt and kubectl. Then install golangand minikube.

### Install virt and kvm

```
sudo dnf install -y qemu-kvm qemu-img libvirt libvirt-python libvirt-client virt-install virt-viewer bridge-utils
sudo systemctl start libvirtd
sudo systemctl enable libvirtd
```

[Documented in more detail here:](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

#### Install kubectl

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

### Install golang

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
```
```
cd ~/go/src/hello
go build
./hello
```

### Install minikube

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

#### Install kvm2 driver

```
curl -LO https://storage.googleapis.com/minikube/releases/latest/docker-machine-driver-kvm2 && chmod +x docker-machine-driver-kvm2 && sudo mv docker-machine-driver-kvm2 /usr/local/bin/
```
     
To check your GID membership for libvirt and docker
```
id $(whoami)
```
start minikube

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
You should see
```
    Id    Name                           State
    ----------------------------------------------------
    1     minikube                       running
```

### Install protobuf
```
sudo dnf install -y protobuf dep
sudo dnf install -y protoc-gen-go
sudo dnf install -y protobuf-compiler
```

## Install and test network service mesh

### Download Network Service Mesh Code

if GOPATH is set as environment variable

```
cd $GOPATH/src
go get github.com/ligato/networkservicemesh
```
## Now start network service mesh!

### Use make from the top level to generate the code, build docker images and install into minikube.
```
cd $GOPATH/src/github.com/ligato/networkservicemesh
make
```
### Alternatively follow the steps manually.

#### Generate deep copy functions, clientset, listeners and informers

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
Verify the generated code.

```
./scripts/verify-codegen.sh
```

#### Build the docker images
```
cd $GOPATH/src/github.com/ligato/networkservicemesh
docker build -t ligato/networkservicemesh/netmesh-test -f build/nsm/docker/Test.Dockerfile .
```
## Package Versions
Everything is from fedora 28 repos except for docker, kubernetes and minikube.
* TODO - List all package versions



