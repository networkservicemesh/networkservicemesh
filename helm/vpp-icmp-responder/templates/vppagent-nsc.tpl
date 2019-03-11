---
apiVersion: extensions/v1beta1
kind: Deployment
spec:
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
          imagePullPolicy: IfNotPresent
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
  namespace: default
