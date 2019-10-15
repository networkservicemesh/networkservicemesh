apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-entries
  namespace: {{ .Values.namespace }}
data:
  registration.json: |-
{{ .Files.Get "registration.json" | indent 4}}
