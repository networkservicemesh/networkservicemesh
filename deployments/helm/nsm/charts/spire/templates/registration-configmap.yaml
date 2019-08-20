apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-entries
data:
  registration.json: |-
{{ .Files.Get "registration.json" | indent 4}}