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

Example run command
-------------------
You will need some configuration files. Two of them are checked into the
repository, and one you can get from your Kubernetes cluster. See the example
run command below.

```
go run cmd/nsm/main.go -kube-config="/Users/kmestery/.kube/config" -etcdv3-config="/Users/kmestery/go/src/github.com/ligato/networkservicemesh/cmd/nsm/etcdv3.conf" -http-config="/Users/kmestery/go/src/github.com/ligato/networkservicemesh/cmd/nsm/http.conf" -microservice-label="netmesh"
```

[1]: http://ligato.io
[2]: https://github.com/ligato/cn-infra
[3]: https://github.com/coreos/etcd/releases/
