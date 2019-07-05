# Configure Network Service Mesh with Kernel-based forwarding plane

The default forwarding plane in Network Service Mesh is VPP.
The following presents an alternative forwarding plane that leverages the built-in tools provided by the Kernel.

Disclaimer:
Note that it is still under development. For example, part of the integration tests are not yet compliant.

## How to configure

To configure Network Service Mesh with the Kernel-based forwarding plane, you can use the following environment variables:

For example:

```bash
FORWARDING_PLANE=kernel-forwarder make k8s-save
```

* FORWARDING_PLANE stands for the forwarding plane that we want to use.

## Give it a try

To try it yourself, you can clone the project and do the following:

Create your Kubernetes cluster using Vagrant:

```bash
FORWARDING_PLANE=kernel-forwarder make vagrant-start
```

Build and save all the necessary images:

```bash
FORWARDING_PLANE=kernel-forwarder make k8s-save
```

Deploy Network Service Mesh:

```bash
FORWARDING_PLANE=kernel-forwarder make k8s-deploy
```

And finally verify that it works as expected:

```bash
FORWARDING_PLANE=kernel-forwarder make k8s-check
```
