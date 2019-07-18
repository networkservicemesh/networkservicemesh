Interdomain NSM
============================

Specification
-------------

Interdomain provides an ability for the Client in one domain to consume a Network Service provided by an Endpoint in another domain.

Implementation details
---------------------------------

1. First, the Client asking for a Network Service with a name of the form: "network-service@example.com"

2. Then the NSMgr sends the registry calls to find the NetworkService Endpoints for the Network Service to the proxy NSMgr in that domain.

3. Proxy NSMgr resolves domain name via DNS and sends a GRPC Request to the remote domain NSMgr to find NetworkService Endpoints for the Network Service

4. Client's NSMgr negotiate the 'remote.Mechanism' with remote domain NSMgr through the proxy NSMgr in Client's domain.

5. Both sides setup their end of the tunnel.
    *  Client's NSMgr initiate connection monitoring via proxy NSMgr for the death and healing purposes

Client and endpoint nodes has to have public ipv4 addresses and have to be reachable by each other.

All remote mechanisms supported by NSM are also suported by Interdomain NSM.

Interdomain NSM does not have central registry. All clusters are communicate just within each single connection.

Network service can be reached by ipv4 format address and domain name. Currently domain name will be resolved by local DNS resolver and can be changed to any custom resolver ([func ResolveDomain(remoteDomain string)](../../k8s/pkg/utils/interdomainutils.go)).

Example usage
------------------------

Following is an example of the full Proxy NSMgr deployment.

```yaml
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: proxy-nsmgr
  namespace: nsm-system
spec:
  selector:
    matchLabels:
      app: proxy-nsmgr-daemonset
  template:
    metadata:
      labels:
        app: proxy-nsmgr-daemonset
    spec:
      containers:
        - name: proxy-nsmd
          image: networkservicemesh/proxy-nsmd
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 5006
              hostPort: 5006
        - name: proxy-nsmd-k8s
          image: networkservicemesh/proxy-nsmd-k8s
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: 80
              hostPort: 5005
          env:
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
---
apiVersion: v1
kind: Service
metadata:
  name: pnsmgr-svc
  labels:
    app: proxy-nsmgr-daemonset
  namespace: nsm-system
spec:
  ports:
    - name: pnsmd
      port: 5005
      protocol: TCP
    - name: pnsr
      port: 5006
      protocol: TCP
  selector:
    app: proxy-nsmgr-daemonset

```

Also take a look an example in interdomain integration tests

Interdomain NSM supports and have been checked on Packet, AWS and GKE clusters. 

References
----------

* Issue(s) reference - https://github.com/networkservicemesh/networkservicemesh/issues/714
* PR reference - https://github.com/networkservicemesh/networkservicemesh/pull/1298