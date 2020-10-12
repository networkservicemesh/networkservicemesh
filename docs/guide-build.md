# Network Service Mesh - Build Guide

## Prerequisites

Make sure you have the following dependencies to run NSM:
* A Kubernetes Cluster - good options include:
* * [kind](guide-kind.md) - usually the easiest choice
* * [vagrant](vagrant/guide-vagrant.md) - useful if you need to debug at the Node Level
* * [gke](guide-gke.md)
* * [azure](guide-azure.md)
* * [aws](guide-aws.md)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [helm](https://helm.sh/)

In addition, to build NSM you will need:

* [Go 1.13 or later](https://golang.org/dl/)
* [Docker](https://docs.docker.com/install/)
* GNU make

## Build

You can build all of the containers needed for NSM, including a bunch of handle Network Service Endpoints (NSEs) and NSCs (Network Service Clients) that are useful for testing, but not part of the core with:

```bash
make k8s-build
```

And if you are using the Kind machinery to run your K8s cluster (described a bit further down), you really want to use the following:

```bash
make k8s-save
```

because ```make k8s-save``` will build your containers and save them in `scripts/vagrant/images` where they can be loaded by the Kind K8s cluster.

You can also selectively rebuild any component, say the `nsmd`, with ```make k8s-nsmd-save```

After installing you can verify it with `helm version`.

## Install

Network Service Mesh provides a handy [Kind](https://github.com/kubernetes-sigs/kind) setup for running a small K8s cluster. Once you've done ```make k8s-save```, you can deploy to it with:

```bash
make k8s-save                                                # build and save the NSM docker containers
make kind-start                                              # start up an nsm cluster named kind
make k8s-load-images                                         # load NSM docker containers into kind
make helm-init                                               # initialize helm
make helm-install-nsm                                        # install the nsm infrastructure
```

## Run
* [icmp-responder](examples/icmp-responder.md) - A simple example that connects an App Pod Client to a Network Service.  
```bash
make helm-install-endpoint helm-install-client
```
* [vpp-icmp-responder](examples/vpp-icmp-example.md) - A simple example that connects a vpp based Pod to a Network Service using memif.  
```bash
make helm-install-vpp-icmp-responder
```
* [vpn](examples/vpn.md) - An example that simulates an App Pod Client connecting to a Network Service implemented as a chain simulating a [VPN Use Case](https://docs.google.com/presentation/d/1Vzmhv5vc10NyAa08ny-CCbveo0_fWkDckbkCD_N0fPg/edit#slide=id.g49bd4e8739_0_12)  
```bash
make helm-install-vpn
```

## Verify
There are set of checkers that allow to verify examples.
* _icmp-responder_ and _vpp-icmp-responder_
    ```bash
    make k8s-icmp-check
    ```
* _vpn_  
    ```bash
    make k8s-vpn-check
    ```

## Uninstall

You can remove the effects of helm-install-% with:

```bash
make helm-delete-%
```

## Integration Testing

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

## Regenerating code

If you change [types.go](https://github.com/networkservicemesh/networkservicemesh/blob/master/k8s/pkg/apis/networkservice/v1alpha1/types.go) or any of the .proto files you will need to be able to run ```go generate ./...``` to regenerate the code.

For rerunning the code generation the required dependencies are retrieved with the script:

```bash
./scripts/prepare-generate.sh
```

To regenerate code:

```bash
go generate ./...
```

**NOTE:**  The script `scripts/install-protoc.sh` will download a released version of `protoc`, however,
at the time of this writing there are no `protoc` releases built with the `grpc` plugin functionality
made use of by the `networkservicemesh` project.  Specifically, the `UnimplementedServer*` method
generation is missing.

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

To install it run:
```bash
make install-formatter
```

To do formatting run:
```bash
make format
```

It may be useful to have `goimports -w -local github.com/networkservicemesh/networkservicemesh` installed as on save hook in your editor. [Go imports doc page](https://godoc.org/golang.org/x/tools/cmd/goimports) may help you to achieve this.

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
Checking changes with memory limitation:
```bash
GOGC=30 make lint-check-diff
```
Checking all code in the project:
```bash 
make lint-check-all
```
If you have any unsolvable problem with a concrete linter then consider updating `.golanci.yaml` 
