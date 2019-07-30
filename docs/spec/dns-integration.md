DNS Integration for NSM
============================

Specification
-------------

Network Service Mesh needs to be able to provide a workload with DNS service from Network Services Without breaking K8s DNS

Implementation details (optional)
---------------------------------

#### nsm-coredns
`nsm-corends` is a docker image based on [coredns](https://github.com/coredns/coredns.io/blob/master/content/manual/what.md). The difference with the original `coredns` in the set of plug-ins. 
The image uses only next `coredns` plugins:
* `bind`
* `hosts`
* `log`
* `reload`

Also, it includes special custom plugin `fanout` (see below).	
#### Fanout plugin
`fanout` is custom [plugin for coredns](https://coredns.io/manual/plugins/).
The fanout plugin re-uses already opened sockets to the upstreams. It supports TCP and DNS-over-TLS and uses in-band health checking. 
For each incoming DNS query that hits the CoreDNS fanout plugin, it will be replicated in parallel to each listed IP. The first non-negative response from any of the queried DNS Servers will be forwarded as a response to the request.

#### How use nsm-coredns as default name server for pod?
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
...
```
Example usage (optional)
------------------------

* TestBasicDns

References
----------
* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/1224
