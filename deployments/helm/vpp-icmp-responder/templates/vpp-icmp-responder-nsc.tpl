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
            - name: CLIENT_LABELS
              value: "app=icmp"
            - name: CLIENT_NETWORK_SERVICE
              value: "icmp-responder"
          resources:
            limits:
              networkservicemesh.io/socket: 1
metadata:
  name: vpp-icmp-responder-nsc
  namespace: {{ .Release.Namespace }}
