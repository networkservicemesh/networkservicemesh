---
apiVersion: extensions/v1beta1
kind: Deployment
spec:
  replicas: 1
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "vpn-gateway-nsc"
        networkservicemesh.io/impl: "secure-intranet-connectivity"
    spec:
      nodeSelector:
        node-role.kubernetes.io/master: ""
      containers:
        - name: alpine-img
          image: alpine:latest
          imagePullPolicy: {{ .Values.pullPolicy }}
          command: ['tail', '-f', '/dev/null']
metadata:
  name: vpn-gateway-nsc
  namespace: default
  annotations:
    ns.networkservicemesh.io: secure-intranet-connectivity
