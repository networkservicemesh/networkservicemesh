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
      serviceAccount: nsc-acc
      containers:
        - name: vppagent-nsc
          image: {{ .Values.registry }}/{{ .Values.org }}/vpp-test-common:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: TEST_APPLICATION
              value: "vppagent-nsc"
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
  annotations:
    security.networkservicemesh.io: ""
