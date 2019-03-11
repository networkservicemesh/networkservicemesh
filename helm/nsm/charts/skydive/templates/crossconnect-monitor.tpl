---
apiVersion: extensions/v1beta1
kind: Deployment
spec:
  template:
    metadata:
      labels:
        app: nsmd-ds
    spec:
      containers:
        - name: crossconnect-monitor
          image: {{ .Values.registry }}/networkservicemesh/crossconnect-monitor:{{ .Values.tag }}
          imagePullPolicy: IfNotPresent
metadata:
  name: crossconnect-monitor
  namespace: default
