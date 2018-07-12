# Running Network Service Mesh

This document covers how to run NSM (Network Service Mesh) in an existing Kubernetes cluster.

## Requirements

* Kubernetes Cluster - minikube may be used for development
* Privileged access to Kubernetes API

## Example run command
You need two configuration files:

* kube.conf: configuration file used to access your kubernetes cluster.
* http.conf: REST API configuration file.

NSM provides a sample REST API configuration file at [/cmd/netmesh/http.conf](/cmd/netmesh/http.conf). Copy and modify http.conf as appropriate. 

## Run as a single container for development/testing

To run netmesh as a single container, execute the following command:

```
docker run -it -p 0.0.0.0:9191:9191 \
--name=netmesh \
-v $HOME/kube.conf:/conf/kube.conf \
-v $HOME/etcdv3.conf:/conf/etcdv3.conf \
-v $HOME/http.conf:/conf/http.conf \
ligato/networkservicemesh/nsm
```

## Run NSM as a daemonset for production
The preferred method for running NSM in a cluster is to deploy a daemonset. A daemonset may be used to run and manage a container designed to run on every node.

To deploy NSM as a daemonset, use the following command:
```
$ kubectl create -f conf/netmesh.yaml 
daemonset "netmesh" created
$ 
```

Label your nodes, first by getting a list of your nodes, and secondly, by labeling the nodes you want to run nsm on:

```
$ kubectl get nodes
NAME        STATUS    AGE
10.0.10.2   Ready     75d
```

Label the nodes:

```
$ kubectl label node 10.0.10.2 app=networkservice-node
node "10.0.10.2" labeled
```

Check your pods, and you'll see the daemonset running:

```
$ kubectl get pods
NAME                        READY     STATUS    RESTARTS   AGE
netmesh-7nfxo               1/1       Running   0          19s
```
