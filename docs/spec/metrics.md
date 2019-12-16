
Metrics in MonitorCrossConnects 
============================

Specification
-------------

Currently, nsmd is sending NSM cross-connection events (UPDATE/DELETE) over a gRPC stream.
Those information contain interfaces that compose cross connections, be it KERNEL, MEMIF, VXLAN…
We’d like to get metrics on those interfaces, such as received and transmitted number of packets and bytes.
Those metrics could be reported periodically over the gRPC stream.

With those information, one is able to monitor bandwidth consumption of each NSM cross connect and troubleshoot an unhealthy cross-connections.

Example usage
------------------------

In order to enable metrics, you need to set the following environment variable:
```
METRICS_COLLECTOR_ENABLED=true
```

Apply it to `helm-install`, f.e.

```
METRICS_COLLECTOR_ENABLED=true make helm-install-nsm helm-install-crossconnect-monitor helm-install-icmp-responder
```

If you want to track the metrics in Prometheus, you need to apply analogically
```
PROMETHEUS=true
```

Port-forward the prometheus-server pod to observe metrics for pod to pod connection:
```
export POD_NAME=$(kubectl get pods --namespace nsm-system -l "app=prometheus-server" -o jsonpath="{.items[0].metadata.name}")
kubectl port-forward $POD_NAME 9090:9090
```
and open Prometheus server on `localhost:9090'.

In order to view the values, you need to use the following queries:
```
rx_bytes{src_pod="<pod1>", src_namespace="<pod1_namespace>", dst_pod="<pod2>", dst_namespace="<pod2_namespace>"}
rx_packets{src_pod="<pod1>", src_namespace="<pod1_namespace>", dst_pod="<pod2>", dst_namespace="<pod2_namespace>"}
rx_error_packets{src_pod="<pod1>", src_namespace="<pod1_namespace>", dst_pod="<pod2>", dst_namespace="<pod2_namespace>"}
tx_bytes{src_pod="<pod1>", src_namespace="<pod1_namespace>", dst_pod="<pod2>", dst_namespace="<pod2_namespace>"}
tx_packets{src_pod="<pod1>", src_namespace="<pod1_namespace>", dst_pod="<pod2>", dst_namespace="<pod2_namespace>"}
tx_error_packets{src_pod="<pod1>", src_namespace="<pod1_namespace>", dst_pod="<pod2>", dst_namespace="<pod2_namespace>"}
```

References
----------

* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/715
* PR reference - https://github.com/networkservicemesh/networkservicemesh/pull/980
