DNS Integration for NSM
============================

Specification
-------------

Network Service Mesh needs to be able to provide a workload with DNS resolution for Network Services without breaking the DNS resolution K8s natively provides.

Overview
--------

The DNS integration capability is performed by running a DNS server co-resident with the application pod that can direct requests to multiple DNS servers. The DNS servers that should be sent requests is controlled by the NSEs that create connections into that application pod.  If the NSEs populate the DNSContext with a DNS server appropriate for this connection then additional containers in the application pod will ensure those DNS servers are consulted. The additional containers are nsm-coredns and nsm-dns-monitor.  They can be inserted directly or via the admission webhook.  The nsm-coredns container is responsible for handling the DNS resolution mechanics. The nsm-dns-monitor container is responsible for updating the configuration that the nsm-coredns container acts on based on DNS information in the connection contexts provided by the NSEs.

The NSM provided SDK provides a capability to read environmental varibles to determine additional DNS servers and domains that should be added to the DNScontext.  These variables are: DNS_SEARCH_DOMAINS and DNS_SERVER_IPS.

Note:  Some of the additional DNS configuration capabilites K8s offers are overridden adn not availabel when these additional containers are inserted.  

Implementation details (optional)
--------------------------------- 

### nsm-coredns
`nsm-corends` is a docker image based on [coredns](https://github.com/coredns/coredns.io/blob/master/content/manual/what.md). The difference with the original `coredns` is the set of plug-ins. 
The image uses only these `coredns` plugins:
* `bind`
* `hosts`
* `log`

Also, it includes a `fanout` plugin defined in the NSM tree (see below).	
### Fanout plugin
`fanout` is a custom [plugin for coredns](https://coredns.io/manual/plugins/).
The fanout plugin re-uses already opened sockets to the upstreams. It supports TCP and DNS-over-TLS and uses in-band health checking.  The config provided to nsm-coredns may include multiple IPs based on the services a pod attachs to. 
Each incoming DNS query that hits the CoreDNS fanout plugin will be replicated in parallel to each listed IP (i.e. the DNS servers). The first non-negative response from any of the queried DNS Servers will be forwarded as a response to the application's DNS request.

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
To dynamically update a `Network Service Client's` `DNSConfigs` based on connections, you could use nsm-dns-monitor. For example:
```
func main() {
        ...
    app := nsmmonitor.NewNSMMonitorApp(common.FromEnv())
    app.SetHandler(nsmmonitor.NewNsmDNSMonitorHandler())
    app.Run()
        ...
}
``` 
Make sure that your application pod includes the `nsm-coredns` and `nsm-coredns` containers and has [environment variable](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/env.md) `USE_UPDATE_API=true`.
See an example of usage `nsm-dns-monitor` in `test/applications/cmd/monitoring-dns-nsc`

### Using nsm-coredns and nsm-dns-monitor without changing the client's deployment configuration
To inject the `nsm-coredns` and `nsm-dns-monitor` containers into a client's pod during deployment, you can simply deploy the [admission webhook](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/spec/admission.md). `Admission webhook` will automatically append the DNS specific containers to your `Network Service Client`.  When using the admission webhook there is no way to disable the insertion of these additional containers. 

## NSE Requirements
In order for the application pod to try multiple DNS servers the NSEs must populate the DNScontext.   The SDK provides a means to populate the DNScontext based on environmental variables provided to the NSE container. These variables are: DNS_SEARCH_DOMAINS and DNS_SERVER_IPS.  The [icmp-responer](test/applications/cmd/icmp-responder-nse/main.go) has an example implementation. 


Disabling NSM DNS resolution
----------------------------
TODO

Example usage (optional)
------------------------

* TestBasicDns
* TestNsmCorednsNotBreakDefaultK8sDNS
* TestDNSMonitoringNsc
* test/applications/cmd/monitoring-dns-nsc

References
----------
* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/1224
