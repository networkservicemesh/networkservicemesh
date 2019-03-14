---
apiVersion: extensions/v1beta1
kind: Deployment
spec:
  template:
    metadata:
      labels:
        app: nsmgr-daemonset
    spec:
      containers:
        - name: crossconnect-monitor
          image: {{ .Values.registry }}/networkservicemesh/crossconnect-monitor:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
metadata:
  name: crossconnect-monitor
  namespace: default
