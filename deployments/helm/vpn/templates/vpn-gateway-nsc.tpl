---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "vpn-gateway-nsc"
      networkservicemesh.io/impl: "secure-intranet-connectivity"
  replicas: 1
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "vpn-gateway-nsc"
        networkservicemesh.io/impl: "secure-intranet-connectivity"
    spec:
      serviceAccount: nsc-acc
      containers:
        - name: alpine-img
          image: alpine:latest
          imagePullPolicy: {{ .Values.pullPolicy }}
          command: ['tail', '-f', '/dev/null']
metadata:
  name: vpn-gateway-nsc
  namespace: {{ .Release.Namespace }}
  annotations:
    ns.networkservicemesh.io: secure-intranet-connectivity
    security.networkservicemesh.io: ""
