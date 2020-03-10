DNS integration for NSM
============================

Specification
-------------

Network Service Mesh needs to be able to provide a workload with DNS resolution for Network Services without breaking the DNS resolution K8s natively provides.

Overview
--------

The DNS integration capability is performed by running a DNS server co-resident with the application pod that can direct requests to multiple DNS servers. The DNS servers that should be sent requests are controlled by the NSEs that create connections into that application pod.  If the NSEs populate the DNSContext with a DNS server appropriate for this connection then additional containers in the application pod will ensure those DNS servers are consulted. The additional containers are coredns and nsm-dns-monitor.  They can be inserted directly or via the admission webhook.  The coredns container is responsible for handling the DNS resolution mechanics. The nsm-dns-monitor container is responsible for updating the configuration that the coredns container acts on based on DNS information in the connection contexts provided by the NSEs.

The NSM endpoint SDK provides two functions `NewAddDNSConfigs` and  `NewAddDnsConfigDstIp` to update the connection context so the nsm-dns-monitor can update the coredns configuration.

Implementation details (optional)
---------------------------------

### Using coredns as the default name server for the pod
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
        forward {IP addresses}
        ...
    }
```
2) Deploy coredns. You can deploy it as sidecar for your Pod (see below).
```
...
spec:
    spec:
      containers:
        #containers...
        - name: coredns
          image: networkservicemesh/coredns:lateest
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
Make sure that your application pod includes the `coredns` and `coredns` containers and has [environment variable](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/env.md) `USE_UPDATE_API=true`.
See an example of usage `nsm-dns-monitor` in `test/applications/cmd/monitoring-dns-nsc`

### Using coredns and nsm-dns-monitor without changing the client's deployment configuration
To inject the `coredns` and `nsm-dns-monitor` containers into a client's pod during deployment, you can simply deploy the [admission webhook](https://github.com/networkservicemesh/networkservicemesh/blob/master/docs/spec/admission.md). `Admission webhook` will automatically append the DNS specific containers to your `Network Service Client`.  When using the admission webhook there is no way to disable the insertion of these additional containers.

## NSE Requirements
In order for the application pod to try multiple DNS servers the NSEs must populate the DNScontext.
The SDK provides functions that the NSE can call to populate the DNScontext.  An example of how this is done using environmental variables is available here: [icmp-responder](test/applications/cmd/icmp-responder-nse/main.go). The environmental variables used are DNS_SEARCH_DOMAINS and DNS_SERVER_IPS.

Example usage (optional)
------------------------

* TestBasicDns
* TestNsmCorednsNotBreakDefaultK8sDNS
* TestDNSMonitoringNsc
* test/applications/cmd/monitoring-dns-nsc

References
----------
* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/1224
