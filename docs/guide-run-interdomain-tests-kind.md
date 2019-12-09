# Using `kind` clusters for run interdomain tests

For run [interdomain](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/spec/interdomain.md) tests at first you need to install  [kind](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/guide-kind.md).

## Create `kind` clusters

Create the first cluster:

``` bash
kind create cluster kind create cluster --name cluster1 --config scripts/kind.yaml 
```

Create the second cluster:

``` bash
kind create cluster kind create cluster --name cluster2 --config scripts/kind.yaml 
```

Make sure that all clusters have been created:

```bash
kubectl  config get-contexts
```

Output should contain created clusters:

```bash
...
CURRENT   NAME            CLUSTER        AUTHINFO       NAMESPACE
          kind-cluster1   kind-cluster1  kind-cluster1   
*         kind-cluster2   kind-cluster2  kind-cluster2
...
```

### Setup `kind` clusters

Configure a cluster:

```bash
kubectl config use-context kind-cluster1
CLUSTER_RULES_PREFIX=kind KIND_CLUSTER_NAME=cluster1 make k8s-config
```

Load images to a cluster:

```bash
CLUSTER_RULES_PREFIX=kind KIND_CLUSTER_NAME=cluster1 make k8s-load-images
```

if you also want to use helm deployment on cluster than you also need to install `helm`:

```
kubectl config use-context kind-cluster1
make helm-install
...
```

NOTE: Make the same things for the second cluster.

### Setup env for interdomain tests

Add env for a process that will run interdomain tests:

```
KUBECONFIG_CLUSTER_1=PATH_TO_KIND_KUBECONFIG1
KUBECONFIG_CLUSTER_2=PATH_TO_KIND_KUBECONFIG2
INSECURE=true
```

For creating kind kubeconfig file you can use:
```
KIND_CLUSTER_NAME=cluster1 make kind-export-config
```
It will create kubeconfig file `cluster1-kubeconfig`.

After that, you can run any interdomain test.