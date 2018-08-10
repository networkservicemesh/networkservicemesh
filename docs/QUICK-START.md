# Quick Start Network Service Mesh

This document assumes you already have a kubernetes cluster, and all packages such as golang, docker and kubernetes. If not,
[How to stand up a VM from scratch with everything for NSM](/docs/complete-startup-guide.md)


### Create the daemonset

```
kubectl create -f conf/sample/networkservice-daemonset.yaml

```

Now you should be able to see your Network Service Mesh daemonset running:

```
kubectl get pods
```
You should see:
```
NAME                   READY     STATUS    RESTARTS   AGE
networkservice-x5k9s   1/1       Running   0          5h
```
