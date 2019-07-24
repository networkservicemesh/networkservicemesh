# Network Service Mesh - Build Guide

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

## Build

All of the actual code in Network Service Mesh builds as pure go:

```bash
go generate ./...
go build ./...
```

But to really do interesting things in NSM, you will want to build various Docker containers, and deploy them to K8s.
All of this is doable via normal Docker/K8s commands, but to speed development, some make machinery has been added to make things easy.

## Building and Saving container images using the Make Machinery

You can build all of the containers needed for NSM, including a bunch of handle Network Service Endpoints (NSEs) and NSCs (Network Service Clients) that are useful for testing, but not part of the core with:

```bash
make k8s-build
```

And if you are using the Vagrant machinery to run your K8s cluster (described a bit further down), you really want to use the following:

```bash
make k8s-save
```

because ```make k8s-save``` will build your containers and save them in `scripts/vagrant/images` where they can be loaded by the Vagrant K8s cluster.

> You can also selectively rebuild any component, say the `nsmd`, with ```make k8s-nsmd-save```

## Running the NSM code

Network Service Mesh provides a handy Vagrant setup for running a two node K8s cluster. Once you've done ```make k8s-save```, you can deploy to it with:

```bash
make k8s-deploy
```

By default this will:

1. Spin up a two node K8s cluster from `scripts/vagrant` if one is not already running.
2. Delete old instances of NSM config if present
3. Load all images from `scripts/vagrant/images` into the `master` and `worker` node
4. Deploy the `nsmd` and `vppagent-dataplane` Daemonsets
5. Deploy a variety of Network Service Endpoints and Network Service Clients
6. Deploy the `crossconnect-monitor` (a useful tool for debugging)

You can check to see things are working properly by typing:

```bash
make k8s-check
```

which will try pinging from NSCs to NSEs.

You can remove the effects of k8s-deploy with:

```bash
make k8s-delete
```

As in the case with `save` and `build`, you can always do this for a particular component, like ```make k8s-nsc-deploy``` or ```make k8s-nsc-delete```.

## Having more control over the deployment

The described quick start method works for fast deployments and quick tests. However, the build infrastructure provides a fine-grained control over the deployments.

### Working with the Vagrant setup

To spin the default 2 node Vagrant setup with Kubernetes on top, type:

```bash
make vagrant-start
```

At any point, you can ```make vagrant-suspend``` and ```make vagrant-resume``` to pause and restore the spawn virtual nodes. If for some reason you need to rebuild or completely destroy the Vagrant environment, use ```make vagrant-restart``` and ```make vagrant-destroy```

To point your host ```kubectl``` to the Kubernetes deployment in the virtual nodes, use:

```bash
source scripts/vagrant/env.sh
```

### Deploying the NSM infrastructure

Network Service Mesh consists of a number of system pods, which take care of service registration, provide the dataplane functionality, do monitoring and observability.

Once you have configured your ```kubectl``` to the desired Kubernetes `master` (may or may not be set through Vagrant), you can initiate the NSM infrastructure deployment and deletion using ```make k8s-infra-deploy``` and ```make k8s-infra-delete```.

### Deploying the ICMP example and testing it

The project comes with a simple, ready to test ICMP example. It deploys a number of ICMP responder NSEs and connects NSCs to them. This shows same and cross-node communication and is good for visualising it with the provided monitoring tools.

The commands to deploy and delete it are ```make k8s-icmp-deploy``` and ```make k8s-icmp-delete```. Checking the operability of the ICMP example is done through ```make k8s-check```

### Deploying the VPN composed Network Service

One of the big advantages on Network Service Mesh is NS composition, i.e. forming a complex service out of a number of simple NSEs. The project comes with an example that implements the "secure-intranet-connectivity" Network Service which connects together a simple ACL based packet filtering firewall and a simulated VPN gateway NSEs.

Deploying it is done through ```make k8s-vpn-deploy``` and to uninstall it - run ```make k8s-vpn-delete```. Checking VPN's operability is done with ```make k8s-check```.

## Trigger the integration tests on your host

You can verify your changes by triggering the integration tests on your host. To do so, execute the following:

If you haven't already, prepare the Vagrant environment:

```bash
make vagrant-start
source scripts/vagrant/env.sh
```

Build the images:

```bash
make k8s-build
```

Load the images:

```bash
make k8s-load-images
```

Trigger all integration tests:

```bash
make k8s-integration-tests
```

or one by one using the test name. For example, to trigger `TestExec`, run:

```bash
make k8s-integration-TestExec-test
```

## Helpful Logging tools

In the course of developing NSM, you will often find yourself wanting to look at logs for various NSM components.

The following:

```bash
make k8s-nsmd-logs
```

will dump all the logs for all running `nsmd` Pods in the cluster (you are going to want to redirect these to a file).

This works for any component in the system.

For example:

```bash
make k8s-crossconnect-monitor-logs
```

dumps the logs from the `crossconnect-monitor`, which has been logging new crossconnects as they come into existence and go away throughout
the cluster.

## Regenerating code

If you change [types.go](https://github.com/networkservicemesh/networkservicemesh/blob/master/k8s/pkg/apis/networkservice/v1alpha1/types.go) or any of the .proto files you will need to be able to run ```go generate ./...``` to regenerate the code.

In order to be able to do that, you need to have installed:

* protobuf - run ```./scripts/install-protoc.sh```
* proto-gen-go - run ```go install ./vendor/github.com/golang/protobuf/protoc-gen-go/```
* deep-copy-gen - run ```go install ./vendor/k8s.io/code-generator/cmd/deepcopy-gen/```

Then just run:

```bash
go generate ./...
```

## Updating Deps

If you need to add new dependencies, run:

```bash
go mod tidy
```

## Shellcheck

As part of our CI, we run shellcheck on all shell scripts in the repo.
If you want to run it locally, you need to [install shellcheck](https://github.com/koalaman/shellcheck#installing)

## Canonical source on how to build

The [.circleci/config.yml](https://github.com/networkservicemesh/networkservicemesh/blob/master/.circleci/config.yml) file is the canonical source of how to build Network Service Mesh in case this file becomes out of date.

## Code formatting
We use `goimports` tool since it formats the code in the same style as `go fmt` and organizes imports additionally.

To install `goimports` run:
```bash
GO111MODULE=off go get golang.org/x/tools/cmd/goimports
```

To do formatting run:
```bash
make format
```

It may be useful to have `goimports` installed as on save hook in your editor. [This page](https://godoc.org/golang.org/x/tools/cmd/goimports) may help you to achieve this.

## Static analysis of code
Get code static analyzer tool:
```bash
make lint-install
```
Make sure that tools is installed and can be used from terminal:
```bash
golangci-lint --version
```
If the command above doesn't work make sure the tool exists in `go/bin` directory.

Checking changes:
```bash
make lint-check-diff
```
Checking all code in the project:
```bash 
make lint-check-all
```
If you have any unsolvable problem with a concrete linter then consider updating `.golanci.yaml` 
