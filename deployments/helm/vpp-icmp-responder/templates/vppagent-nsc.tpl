---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io: "true"
      networkservicemesh.io/app: "vppagent-nsc"
  replicas: 2
  template:
    metadata:
      labels:
        networkservicemesh.io: "true"
        networkservicemesh.io/app: "vppagent-nsc"
    spec:
      hostPID: true
      containers:
        - name: vppagent-nsc
          image: {{ .Values.registry }}/networkservicemesh/vppagent-nsc:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: OUTGOING_NSC_LABELS
              value: "app=icmp"
            - name: OUTGOING_NSC_NAME
              value: "icmp-responder"
          resources:
            limits:
              networkservicemesh.io/socket: 1
metadata:
  name: vppagent-nsc
  namespace: {{ .Release.Namespace }}
