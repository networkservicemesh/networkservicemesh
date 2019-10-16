Liveness/Readiness probes
============================

Specification
-------------

Kubernetes provides a mechanism called [Readiness and Liveliness probes](https://cloud.google.com/blog/products/gcp/kubernetes-best-practices-setting-up-health-checks-with-readiness-and-liveness-probes)
to allow it to determine whether a Pod is ‘Ready’ or in need of restart (Liveliness).

The following is a [good example](https://www.ianlewis.org/en/using-kubernetes-health-checks)
of how to write a simple Readiness Probe in Go.

For the time being, the probe checking covers the following services - `nsmgr` and `vppagent-forwarder`.

Implementation details
---------------------------------

* Created `/liveness` and `/readiness` HTTP handlers for both components serving on port `5555` (configurable)
* Each component sets specific flags stating if its dependencies initialized okay or not
* Updated the `kube-testing` description files to take into account these changes
* Updated the Kubernetes configuration files to enable liveness/readiness probing
* Added measuring of the boot time for NSMgr and VPP agent data-plane for debugging
* The timings used for the liveness/readiness probes are more or less the default ones
adjusted to match our Vagrant setup and keep the deployment stable.
* For instance, the default `timeoutSeconds` value of 1 was not suitable for our use case causing
the liveness probe check to fail.

Example usage
------------------------

Leveraging the healthcheck features in Kubernetes is enabled by adding the following to the yaml file:

```bash
          livenessProbe:
            httpGet:
              path: /liveness
              port: 5555
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 3
          readinessProbe:
            httpGet:
              path: /readiness
              port: 5555
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 3
```

References
----------

* Issue(s) reference - [#711](https://github.com/networkservicemesh/networkservicemesh/issues/711)
* PR reference - [#730](https://github.com/networkservicemesh/networkservicemesh/pull/730)
