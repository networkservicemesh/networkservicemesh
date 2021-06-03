# Test kernel to kernel connection and VFIO connection

This example shows that local kernel connection and VFIO connection can be setup by NSM at the same time.

## Run

Create test namespace:
```bash
NAMESPACE=($(kubectl create -f ../namespace.yaml)[0])
NAMESPACE=${NAMESPACE:10}
```

Register namespace in `spire` server:
```bash
kubectl exec -n spire spire-server-0 -- \
/opt/spire/bin/spire-server entry create \
-spiffeID spiffe://example.org/ns/${NAMESPACE}/sa/default \
-parentID spiffe://example.org/ns/spire/sa/spire-agent \
-selector k8s:ns:${NAMESPACE} \
-selector k8s:sa:default
```

Select node to deploy NSC and NSE:
```bash
NODE=($(kubectl get nodes -o go-template='{{range .items}}{{ if not .spec.taints  }}{{index .metadata.labels "kubernetes.io/hostname"}} {{end}}{{end}}')[0])
```

Create customization file:
```bash
cat > kustomization.yaml <<EOF
---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: ${NAMESPACE}

bases:
- ../../../apps/nsc-kernel
- ../../../apps/nse-kernel
- ../../../apps/nsc-vfio
- ../../../apps/nse-vfio

patchesStrategicMerge:
- patch-nsc.yaml
- patch-nse.yaml
EOF
```

Create kernel NSC patch:
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
            - name: NSM_NETWORK_SERVICES
              value: kernel://icmp-responder/nsm-1
      nodeSelector:
        kubernetes.io/hostname: ${NODE}
EOF
```

Create kernel NSE patch:
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

Deploy NSCs and NSEs:
```bash
kubectl apply -k .
```

Wait for applications ready:
```bash
kubectl wait --for=condition=ready --timeout=1m pod -l app=nsc-kernel -n ${NAMESPACE}
```
```bash
kubectl wait --for=condition=ready --timeout=1m pod -l app=nse-kernel -n ${NAMESPACE}
```
```bash
kubectl wait --for=condition=ready --timeout=1m pod -l app=nsc-vfio -n ${NAMESPACE}
```
```bash
kubectl wait --for=condition=ready --timeout=1m pod -l app=nse-vfio -n ${NAMESPACE}
```

Find NSC and NSE pods by labels:
```bash
NSC_KERNEL=$(kubectl get pods -l app=nsc-kernel -n ${NAMESPACE} --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```
```bash
NSE_KERNEL=$(kubectl get pods -l app=nse-kernel -n ${NAMESPACE} --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```
```bash
NSC_VFIO=$(kubectl get pods -l app=nsc-vfio -n ${NAMESPACE} --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```

Prepare VFIO ping function:
```bash
function dpdk_ping() {
  err_file="$(mktemp)"
  trap 'rm -f "${err_file}"' RETURN

  out="$(kubectl -n ${NAMESPACE} exec ${NSC_VFIO} --container pinger -- /bin/bash -c '\
    /root/dpdk-pingpong/build/app/pingpong                                            \
      --no-huge                                                                       \
      --                                                                              \
      -n 500                                                                          \
      -c                                                                              \
      -C 0a:11:22:33:44:55                                                            \
      -S 0a:55:44:33:22:11                                                            \
  ' 2>"${err_file}")"

  if [[ "$?" != 0 ]]; then
    cat "${err_file}" 1>&2
    echo "${out}" 1>&2
    return 1
  fi

  if ! pong_packets="$(echo "${out}" | grep "rx .* pong packets" | sed -E 's/rx ([0-9]*) pong packets/\1/g')"; then
    cat "${err_file}" 1>&2
    echo "${out}" 1>&2
    return 1
  fi

  if [[ "${pong_packets}" == 0 ]]; then
    cat "${err_file}" 1>&2
    echo "${out}" 1>&2
    return 1
  fi

  echo "${out}"
  return 0
}
```

Ping from kernel NSC to kernel NSE:
```bash
kubectl exec ${NSC_KERNEL} -n ${NAMESPACE} -- ping -c 4 172.16.1.100
```

Ping from kernel NSE to kernel NSC:
```bash
kubectl exec ${NSE_KERNEL} -n ${NAMESPACE} -- ping -c 4 172.16.1.101
```

Ping from VFIO NSC to VFIO NSE:
```bash
dpdk_ping
```

## Cleanup

Stop ponger:
```bash
NSE_VFIO=$(kubectl get pods -l app=nse-vfio -n ${NAMESPACE} --template '{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}')
```
```bash
kubectl -n ${NAMESPACE} exec ${NSE_VFIO} --container ponger -- /bin/bash -c '\
  sleep 10 && kill $(pgrep "pingpong") 1>/dev/null 2>&1 &                    \
'
```

Delete ns:
```bash
kubectl delete ns ${NAMESPACE}
```