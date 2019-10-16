DNS Integration for NSM
============================

Specification
-------------

Network Service Mesh needs to be able to provide a workload with DNS service from Network Services Without breaking K8s DNS

Implementation details (optional)
---------------------------------

### nsm-coredns
`nsm-corends` is a docker image based on [coredns](https://github.com/coredns/coredns.io/blob/master/content/manual/what.md). The difference with the original `coredns` in the set of plug-ins. 
The image uses only next `coredns` plugins:
* `bind`
* `hosts`
* `log`

Also, it includes special custom plugin `fanout` (see below).	
### Fanout plugin
`fanout` is custom [plugin for coredns](https://coredns.io/manual/plugins/).
The fanout plugin re-uses already opened sockets to the upstreams. It supports TCP and DNS-over-TLS and uses in-band health checking. 
For each incoming DNS query that hits the CoreDNS fanout plugin, it will be replicated in parallel to each listed IP. The first non-negative response from any of the queried DNS Servers will be forwarded as a response to the request.

### Using nsm-coredns as the default name server for the pod
1) Deploy configmap with corefile content.
```
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns
  namespace: nsm-system
data:
  Corefile: |
    {domain} {
        log
        fanout {IP addresses}
        ...
    }
```
2) Deploy nsm-coredns. You can deploy it as sidecar for your Pod (see below).
```
...
spec:
    spec:
      containers:
        #containers...
        - name: nsm-coredns
          image: networkservicemesh/nsm-coredns:lateest
          imagePullPolicy: IfNotPresent
          volumeMounts: 
            - name: config-volume
              readOnly: true
              mountPath: /etc/coredns
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
            - key: Corefile
              path: Corefile
      DnsConfig:
        nameServers:
          - 127.0.0.1
        searches:
          - default.svc.cluster.local
          - svc.cluster.local
          - cluster.local
        options:
          - ndots: 5
...
```
### nsm-dns-monitor
For add to `Network Service Client` possible to dynamically update `DNSConfigs` from connections, you could use nsm-dns-monitor. For example:
```
func main() {
        ...
    app := nsmmonitor.NewNSMMonitorApp(common.FromEnv())
    app.SetHandler(nsmmonitor.NewNsmDNSMonitorHandler())
    app.Run()
        ...
}
``` 
Make sure that your pod colocated with `nsm-coredns` and `nsm-coredns` has [environment variable](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/env.md) `USE_UPDATE_API=true`.
See at example of usage `nsm-dns-monitor` in `test/applications/cmd/monitoring-dns-nsc`

### Using nsm-coredns and nsm-dns-monitor without changes client's deployment configuration
For injection `nsm-coredns` and `nsm-dns-monitor` containers into client's POD on deploy step, you can simply deploy the [admission webhook](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/spec/admission.md). In this case, you do not need to make any additional changes to the configuration of your deployments. `Admission webhook` will automatically append DNS specific containers to your `Network Service Client`.

Example usage (optional)
------------------------

* TestBasicDns
* TestNsmCorednsNotBreakDefaultK8sDNS
* TestDNSMonitoringNsc
* test/applications/cmd/monitoring-dns-nsc

References
----------
* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/1224
