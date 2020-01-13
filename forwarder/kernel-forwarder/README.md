# Configure Network Service Mesh with Kernel-based forwarding plane

The default forwarding plane in Network Service Mesh is VPP.
The following presents an alternative forwarding plane that leverages the built-in tools provided by the Kernel.

Disclaimer:
Note that it is still under development. For example, part of the integration tests are not yet compliant.

## How to configure

To configure Network Service Mesh with the Kernel-based forwarding plane, you can use the following environment variables:

For example:

```bash
FORWARDING_PLANE=kernel make k8s-save
```

* FORWARDING_PLANE stands for the forwarding plane that we want to use.

## Give it a try

To try it yourself, you can clone the project and do the following:

Create your Kubernetes cluster using Vagrant:

```bash
FORWARDING_PLANE=kernel make vagrant-start
```

Build and save all the necessary images:

```bash
FORWARDING_PLANE=kernel make k8s-save
```

Deploy Network Service Mesh:

```bash
FORWARDING_PLANE=kernel make helm-install-nsm
```

Deploy icmp-responder and nsc:

```bash
FORWARDING_PLANE=kernel make helm-install-endpoint helm-install-client
```

And finally verify that it works as expected:

```bash
FORWARDING_PLANE=kernel make k8s-icmp-check
```

## How it works

As the name implies, the forwarding plane uses the built-in tools provided by the Linux Kernel.

The following are two of the main tools that the kernelforwarder package leverages:

* Netlink - a Linux Kernel interface used for inter-process communication between both the kernel and the userspace processes
* Linux Network Namespaces - provides resource isolation by wrapping a system resource into an abstraction which is bound only to processes within that namespace

The kernelforwarder uses each of these tools through the following Go libraries:

* https://github.com/vishvananda/netlink
* https://github.com/vishvananda/netns

### How to handle local connections

A local connection is when both the client and the endpoint live on the same host.
Since NSM creates point-to-point connections, there's really no point of doing something more complicated on that host to connect the two namespaces.

In this case, we use a pair of VETH interfaces to create the connection. The algorithm is pretty straightforward:

* First, we create the VETH pair
* One side of that pair is injected to the client
* Then, the other one is injected to the endpoint
* Finally, we set names and IPs on both the interfaces

This is implemented in - [local.go](./pkg/kernelforwarder/local.go)

### How to handle remote connections

A remote connection on the other hand is when the client and the endpoint live on different hosts.

In this case, the approach is slightly similar, but this time we use VXLAN interfaces to create the connection. The algorithm is the following:

* On each host, we extract the VXLAN VNI provided by the connection request data and everything else that is related for creating the connection
* Then, we create a VXLAN terminated interface based on that request data - VNI, IPs, etc.
* We inject the interface to the corresponding namespace - client/endpoint
* Finally, we set the name and the IP with the desired ones

This is implemented in - [remote.go](./pkg/kernelforwarder/remote.go)
