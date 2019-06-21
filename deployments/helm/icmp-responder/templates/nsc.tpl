---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "icmp-responder-nsc"
  replicas: 4
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "icmp-responder-nsc"
    spec:
      containers:
        - name: alpine-img
          image: alpine:latest
          command: ['tail', '-f', '/dev/null']
metadata:
  name: alpine-nsc
  namespace: {{ .Release.Namespace }}
  annotations:
    ns.networkservicemesh.io: icmp-responder?app=icmp
