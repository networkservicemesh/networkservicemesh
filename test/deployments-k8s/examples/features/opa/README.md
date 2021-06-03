# Feature OPA

Let's consider a current simplified version of NSM authorization.

![NSM Authorize Scheme](./scheme.png "NSM Authorize Scheme")

*Note: This scheme simplified many of the complex things that happen in every client and endpoint for simplicity. To understand it in deep consider looking at the source code of applications.*

Each application in the path of NSM request doesn't trust anybody. Each endpoint doesn't trust the client and on each incoming request the endpoint validates tokens in the path and if they invalid then the endpoint returns an error.
Each client also doesn't trust the endpoint and checks tokens on the response.

Authorization checks enabled by default in NSM. 
For example, all [use-cases](../../use-cases) are using valid token chains by default. 

The example below will do token from step1 from the scheme as invalid.
Expected that Endpoint(in this case NSMgr) will fail the Request from the client on step 4.

## Run

1. Create test namespace:
```bash
NAMESPACE=($(kubectl create -f ../namespace.yaml)[0])
NAMESPACE=${NAMESPACE:10}
```

2. Register namespace in `spire` server:
```bash
kubectl exec -n spire spire-server-0 -- \
/opt/spire/bin/spire-server entry create \
-spiffeID spiffe://example.org/ns/${NAMESPACE}/sa/default \
-parentID spiffe://example.org/ns/spire/sa/spire-agent \
-selector k8s:ns:${NAMESPACE} \
-selector k8s:sa:default
```

3. Select node to deploy NSC and NSE:
```bash
NODE=($(kubectl get nodes -o go-template='{{range .items}}{{ if not .spec.taints  }}{{index .metadata.labels "kubernetes.io/hostname"}} {{end}}{{end}}')[0])
```

4. Create customization file:
```bash
cat > kustomization.yaml <<EOF
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: ${NAMESPACE}

bases:
- ../../../apps/nsc-kernel
- ../../../apps/nse-kernel

patchesStrategicMerge:
- patch-nsc.yaml
- patch-nse.yaml
EOF
```

5. **Create NSC patch that making any generated token invalid:**
```bash
cat > patch-nsc.yaml <<EOF
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nsc-kernel
spec:
  template:
    spec:
      containers:
        - name: nsc
          env:
            - name: NSM_MAX_TOKEN_LIFETIME
              value: -1m
            - name: NSM_NETWORK_SERVICES
              value: kernel://icmp-responder/nsm-1
      nodeSelector:
        kubernetes.io/hostname: ${NODE}
EOF
```

6. Create NSE patch:
```bash
cat > patch-nse.yaml <<EOF
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nse-kernel
spec:
  template:
    spec:
      containers:
        - name: nse
          env:
            - name: NSE_CIDR_PREFIX
              value: 172.16.1.100/31
      nodeSelector:
        kubernetes.io/hostname: ${NODE}
EOF
```

7. Deploy NSC and NSE:
```bash
kubectl apply -k .
```

8. Wait for applications ready:
```bash
kubectl wait --for=condition=ready --timeout=1m pod -l app=nsc-kernel -n ${NAMESPACE}
```
```bash
kubectl wait --for=condition=ready --timeout=1m pod -l app=nse-kernel -n ${NAMESPACE}
```

9. Find nsc and nse pods by labels:
```bash
NSC=$(kubectl get pods -l app=nsc-kernel -n ${NAMESPACE} --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```
```bash
NSE=$(kubectl get pods -l app=nse-kernel -n ${NAMESPACE} --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

10. Check that NSC is not privileged and it cannot connect to NSE.

```bash
kubectl logs ${NSC} -n ${NAMESPACE} | grep "PermissionDenied desc = no sufficient privileges"
```

## Cleanup

Delete ns:
```bash
kubectl delete ns ${NAMESPACE}
```