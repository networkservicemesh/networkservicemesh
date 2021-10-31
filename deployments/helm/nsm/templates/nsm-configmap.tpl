---
apiVersion: v1
kind: ConfigMap
metadata:
  name: nsm-config
  namespace: {{ .Release.Namespace }}
data:
  excluded_prefixes.yaml: ''
