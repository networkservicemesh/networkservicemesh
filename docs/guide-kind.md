# Using `kind` as a cluster provider

[`kind`](https://kind.sigs.k8s.io/) is a tool for running local Kubernetes clusters using Docker container “nodes”.
Docker is the only prerequisite, it does not require any additional steps, hypervisors etc.

It is worth noting that `kind` as any other Kubernetes deployment tool would expect that the machine that hosts the Docker has at least 4 CPU cores and 4 GB of RAM. That is specifically pointed for OSX users in the official [docs](https://kind.sigs.k8s.io/docs/user/quick-start/).

## Installing `kind`

The default behaviour is to use the installed `kind` version and not update it. An update can be forces by:

```shell
make kind-install
```

## `kind` lifecycle management

To start a `kind` cluster, just run the below command from root networkservicemesh directory:

```shell
$ make kind-start
Creating cluster "nsm" ...
 ✓ Ensuring node image (kindest/node:v1.16.3) 🖼
 ✓ Preparing nodes 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
 ✓ Joining worker nodes 🚜
 ✓ Waiting ≤ 2m0s for control-plane = Ready ⏳
 • Ready after 7s 💚
Set kubectl context to "kind-nsm"
You can now use your cluster with:

kubectl cluster-info --context kind-nsm

Thanks for using kind! 😊
node/nsm-control-plane untainted
```

Using `kubectl` one can verify that the context is set to `kind-nsm`.

```shell
$ kubectl config get-contexts
CURRENT   NAME                 CLUSTER          AUTHINFO         NAMESPACE
          docker-desktop       docker-desktop   docker-desktop
          docker-for-desktop   docker-desktop   docker-desktop
*         kind-nsm             kind-nsm         kind-nsm
```

Deleting the cluster is as easy as:

```shell
$ make kind-stop
Deleting cluster "nsm" ...
```
