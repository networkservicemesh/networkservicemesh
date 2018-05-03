Running Network Service Mesh
============================

Network Service Mesh makes use of some infrastructure provided by [ligato][1],
specifically the [cloud-native infra][2] work.

Requirements
------------
Network Service Mesh makes use of etcd. The simplest way to get an etcd is to
install a Docker container running etcd version 3.3.4. Follow the instructions
found [here][3].

Network Service Mesh requires a kubernetes cluster as well. The details of
installing a Kubernetes cluster are not spelled out here. Network Service Mesh
will work with Kubernetes for Mac, however.

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
You need three configuration files:

* kube.conf: This is the kubernetes configuration file used to access your
  kubernetes cluster.
* etcdv3.conf: This is the etcd configuration file.
* http.conf: The file used to expose the REST API.

The etcdv3.conf and http.conf are checked into the repository in the `cmd/nsm`
directory. Copy them and modify as appropriate, and provide your own kube.conf.

To run nsm, execute a command as below:

```
docker run -it --name=nsm -v /Users/kmestery/kube.conf:/conf/kube.conf -v /Users/kmestery/etcdv3.conf:/conf/etcdv3.conf -v /Users/kmestery/http.conf:/conf/http.conf ligato/networkservicemesh/nsm
```

This will mount the configuration files as volumes into the container.


[1]: http://ligato.io
[2]: https://github.com/ligato/cn-infra
[3]: https://github.com/coreos/etcd/releases/
