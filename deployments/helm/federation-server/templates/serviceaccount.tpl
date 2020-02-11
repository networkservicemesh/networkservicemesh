apiVersion: v1
kind: ServiceAccount
metadata:
  name: federation-server-acc
  namespace: {{ .Release.Namespace }}