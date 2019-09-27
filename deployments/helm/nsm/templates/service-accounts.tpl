apiVersion: v1
kind: ServiceAccount
metadata:
  name: nse-acc
  namespace: {{ .Release.Namespace }}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nsc-acc
  namespace: {{ .Release.Namespace }}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nsmgr-acc
  namespace: {{ .Release.Namespace }}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: forward-plane-acc
  namespace: {{ .Release.Namespace }}