---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "icmp-responder-nsc"
  replicas: 2
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "icmp-responder-nsc"
    spec:
      serviceAccount: nsc-acc
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: networkservicemesh.io/app
                    operator: In
                    values:
                      - icmp-responder-nsc
              topologyKey: "kubernetes.io/hostname"
      containers:
        - name: alpine-img
          image: alpine:latest
          command: ['tail', '-f', '/dev/null']
metadata:
  name: icmp-responder-nsc
  namespace: {{ .Release.Namespace }}
  annotations:
    ns.networkservicemesh.io: {{ .Values.networkservice }}?app=icmp
