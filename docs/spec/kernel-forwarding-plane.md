Kernel-based forwarding plane for Network Service Mesh
============================

Specification
-------------

The default forwarding plane in Network Service Mesh is VPP.
The following presents an alternative forwarding plane that leverages the built-in tools provided by the Kernel.

Disclaimer:
Note that it is still under development. For example, part of the integration tests are not yet compliant.

Implementation details (optional)
---------------------------------

More details about the implementation can be found here - [link to docs](../../dataplane/kernel-forwarder/README.md)

Example usage (optional)
------------------------

To configure Network Service Mesh with the Kernel-based forwarding plane, you can use the following environment variables:

For example:

```bash
FORWARDING_PLANE=kernel make k8s-save
```

* FORWARDING_PLANE stands for the forwarding plane that we want to use.

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

And finally verify that it works as expected:

```bash
FORWARDING_PLANE=kernel make k8s-check
```

References
----------

* Issue(s) reference - [#1169](https://github.com/networkservicemesh/networkservicemesh/issues/1169)
* PR reference - [#1321](https://github.com/networkservicemesh/networkservicemesh/pull/1321)
