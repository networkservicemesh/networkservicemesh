---
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: nsm-role-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: nsm-role
subjects:
  - kind: ServiceAccount
    name: nsmgr-acc
    namespace: {{ .Release.Namespace }}
