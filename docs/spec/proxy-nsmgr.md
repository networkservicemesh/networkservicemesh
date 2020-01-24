Proxy NSMgr
============================

Specification
-------------

Proxy NSMgr provides an ability to pass Network Service requests and connection monitoring via any point of network.

 * Current realization of Proxy NSMgr is a part of the implementation of the Interdomain connectivity

Implementation details
---------------------------------

* Proxy NSMgr starts as kubernetes service. All started Proxy NSMgr instances can be reached by a single DNS name "pnsmgr-svc"

* Client ask for a Network Service with a name of the form "network-service@destination-address" to pass all requests through Proxy NSMgr

* NSMgr sends the registry calls to find the NetworkService Endpoints for the Network Service to the Proxy NSMgr.

* Client's NSMgr negotiate the 'remote.Mechanism' with remote domain NSMgr through the proxy NSMgr.

*  Client's NSMgr initiate connection monitoring via proxy NSMgr for the death and healing purposes

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
            - containerPort: 5005
              hostPort: 80
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

References
----------

* PR reference - https://github.com/networkservicemesh/networkservicemesh/pull/1416

* Interdomain PR reference - https://github.com/networkservicemesh/networkservicemesh/pull/1298