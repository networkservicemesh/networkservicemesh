apiVersion: v1
kind: Secret
metadata:
  name: spire-secret
  namespace: {{ .Values.namespace }}
type: Opaque
data:
  bootstrap.key: |-
{{ .Files.Get "key.pem" | b64enc | indent 4 }}
  bootstrap.crt: |-
{{ .Files.Get "cert.pem" | b64enc | indent 4 }}
