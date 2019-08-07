---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      app: nsmgr-daemonset
  template:
    metadata:
      labels:
        app: nsmgr-daemonset
    spec:
      containers:
        - name: crossconnect-monitor
          image: {{ .Values.registry }}/{{ .Values.org }}/crossconnect-monitor:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
metadata:
  name: crossconnect-monitor
  namespace: {{ .Release.Namespace }}
