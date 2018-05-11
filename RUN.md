Running Network Service Mesh
============================

Network Service Mesh makes use of some infrastructure provided by [ligato][1],
specifically the [cloud-native infra][2] work.

Requirements
------------
Network Service Mesh requires a kubernetes cluster. The details of installing a
Kubernetes cluster are not spelled out here. Network Service Mesh will work
with Kubernetes for Mac, however.

You may need to expose a proxy into your kubernetes cluster. For example, if
you're running Kubernetes for Mac. To do this, run a kubectl command similar to
the following:

```
kubectl proxy --port=8080 --address=<IP ADDRESS>
```

Replace `IP ADDRESS` above with a valid IP address from which the container
running Network Service Mesh can access your kubernetes proxy.

Example run command
-------------------
You need two configuration files:

* kube.conf: This is the kubernetes configuration file used to access your
  kubernetes cluster.
* http.conf: The file used to expose the REST API.

The http.conf file is checked into the repository in the `cmd/nsm`
directory. Copy it and modify as appropriate, and provide your own kube.conf.

Run as a single container
-------------------------

To run netmesh as a single container, execute a command as below:

```
docker run -it -p 0.0.0.0:9191:9191 --name=netmesh -v /Users/kmestery/kube.conf:/conf/kube.conf -v /Users/kmestery/etcdv3.conf:/conf/etcdv3.conf -v /Users/kmestery/http.conf:/conf/http.conf ligato/networkservicemesh/nsm
```

Run netmesh as a daemonset
--------------------------

The more interesting approach is to run netmesh as a daemonset so Kubernetes
takes care of running it on the appropriately labeled nodes. To do this,
run the following commands:

```
$ kubectl create -f conf/netmesh.yaml 
daemonset "netmesh" created
$ 
```

Label your nodes, first by getting a list of your nodes, and secondly by
labeling the nodes you want to run netmesh on:

```
$ kubectl get nodes
NAME        STATUS    AGE
10.0.10.2   Ready     75d
```

Label the nodes:

```
$ kubectl label node 10.0.10.2 app=netmesh-node
node "10.0.10.2" labeled
```

Check your pods and you'll see the daemonset running:

```
$ kubectl get pods
NAME                        READY     STATUS    RESTARTS   AGE
netmesh-7nfxo               1/1       Running   0          19s
```


[1]: http://ligato.io
[2]: https://github.com/ligato/cn-infra
