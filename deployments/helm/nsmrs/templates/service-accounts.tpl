apiVersion: v1
kind: ServiceAccount
metadata:
  name: nsmrs-acc
  namespace: {{ .Release.Namespace }}