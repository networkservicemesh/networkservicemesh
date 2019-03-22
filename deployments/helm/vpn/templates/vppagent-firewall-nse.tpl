---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "firewall"
      networkservicemesh.io/impl: "secure-intranet-connectivity"
  replicas: 1
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "firewall"
        networkservicemesh.io/impl: "secure-intranet-connectivity"
    spec:
      containers:
        - name: firewall-nse
          image: {{ .Values.registry }}/networkservicemesh/vppagent-firewall-nse:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: ADVERTISE_NSE_NAME
              value: "secure-intranet-connectivity"
            - name: ADVERTISE_NSE_LABELS
              value: "app=firewall"
            - name: OUTGOING_NSC_NAME
              value: "secure-intranet-connectivity"
            - name: OUTGOING_NSC_LABELS
              value: "app=firewall"
            - name: TRACER_ENABLED
              value: "true"
          resources:
            limits:
              networkservicemesh.io/socket: 1
metadata:
  name: vppagent-firewall-nse
  namespace: default
