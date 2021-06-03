# Alpine requests for CoreDNS service

This example demonstrates how an external client configures DNS from the connected endpoint. 
Note: NSE provides DNS by itself. Also, NSE could provide configs for any other external DNS servers(that are not located as sidecar with NSE).

## Requires

Make sure that you have completed steps from [features](../)

## Run

Note: Admission webhook is required and should be started at this moment.
```bash
WH=$(kubectl get pods -l app=admission-webhook-k8s -n nsm-system --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
kubectl wait --for=condition=ready --timeout=1m pod ${WH} -n nsm-system
```

1. Create test namespace:
```bash
NAMESPACE=($(kubectl create -f ../../use-cases/namespace.yaml)[0])
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

3. Get all available nodes to deploy:
```bash
NODES=($(kubectl get nodes -o go-template='{{range .items}}{{ if not .spec.taints  }}{{index .metadata.labels "kubernetes.io/hostname"}} {{end}}{{end}}'))
```

4. Create alpine deployment and set `nodeSelector` to the first node:
```bash
cat > alpine.yaml <<EOF
---
apiVersion: v1
kind: Pod
metadata:
  name: alpine
  annotations:
    networkservicemesh.io: kernel://my-coredns-service/nsm-1
  labels:
    app: alpine
spec:
  containers:
  - name: alpine
    image: alpine
    stdin: true
    tty: true
  nodeSelector:
    kubernetes.io/hostname: ${NODES[0]}
EOF
```


5. Add to nse-kernel the corends container and set `nodeSelector` it to the second node:
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
          - name: NSE_SERVICE_NAME
            value: my-coredns-service
          - name: NSE_CIDR_PREFIX
            value: 172.16.1.100/31
          - name: NSE_DNS_CONFIGS
            value: "[{\"dns_server_ips\": [\"172.16.1.100\"], \"search_domains\": [\"my.coredns.service\"]}]"
      - name: coredns
        image: coredns/coredns:1.8.3
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            memory: 170Mi
          requests:
            cpu: 100m
            memory: 70Mi
        args: [ "-conf", "/etc/coredns/Corefile" ]
        volumeMounts:
        - name: config-volume
          mountPath: /etc/coredns
          readOnly: true
        ports:
        - containerPort: 53
          name: dns
          protocol: UDP
        - containerPort: 53
          name: dns-tcp
          protocol: TCP
      nodeSelector:
        kubernetes.io/hostname: ${NODES[1]}
      volumes:
        - name: config-volume
          configMap:
            name: coredns
            items:
            - key: Corefile
              path: Corefile
EOF
```

6. Create kustomization file:
```bash
cat > kustomization.yaml <<EOF
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: ${NAMESPACE}

bases:
- ../../../apps/nse-kernel

resources:
- alpine.yaml
- coredns-config-map.yaml

patchesStrategicMerge:
- patch-nse.yaml
EOF
```

7. Deploy alpine and nse
```bash
kubectl apply -k .
```

8. Wait for applications ready:
```bash
kubectl wait --for=condition=ready --timeout=5m pod alpine -n ${NAMESPACE}
```
```bash
kubectl wait --for=condition=ready --timeout=5m pod -l app=nse-kernel -n ${NAMESPACE}
```

9. Find NSC and NSE pods by labels:
```bash
NSC=$(kubectl get pods -l app=alpine -n ${NAMESPACE} --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```
```bash
NSE=$(kubectl get pods -l app=nse-kernel -n ${NAMESPACE} --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

10. Ping from alpine to NSE by domain-name:`my.coredns.service`
```bash
kubectl exec ${NSC} -c alpine -n ${NAMESPACE} -- ping -c 4 my.coredns.service
```

## Cleanup

Delete ns:
```bash
kubectl delete ns ${NAMESPACE}
```