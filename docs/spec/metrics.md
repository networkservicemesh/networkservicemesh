
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

Take a look at example in test **basic_monitor_crossconnect_metrics_test.go** 

References
----------

* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/715
* PR reference - https://github.com/networkservicemesh/networkservicemesh/pull/980
