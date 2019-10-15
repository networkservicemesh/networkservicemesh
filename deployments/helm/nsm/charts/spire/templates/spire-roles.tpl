apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: spire-agent-role
rules:
  - apiGroups: [""]
    resources: ["nodes/proxy"]
    verbs: ["get", "watch", "list", "create"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: spire-server-role
rules:
  - apiGroups: ["authentication.k8s.io"]
    resources: ["tokenreviews"]
    verbs: ["get", "watch", "list", "create"]
  - apiGroups: [""]
    resources: ["configmaps"]
    resourceNames: ["spire-bundle"]
    verbs: ["get", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
# This cluster role binding allows anyone in the "manager" group to read secrets in any namespace.
kind: ClusterRoleBinding
metadata:
  name: spire-agent-binding
subjects:
  - kind: ServiceAccount
    name: spire-agent
    namespace: {{ .Values.namespace }}
roleRef:
  kind: ClusterRole
  name: spire-agent-role
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: rbac.authorization.k8s.io/v1
# This cluster role binding allows anyone in the "manager" group to read secrets in any namespace.
kind: ClusterRoleBinding
metadata:
  name: spire-server-binding
subjects:
  - kind: ServiceAccount
    name: spire-server
    namespace: {{ .Values.namespace }}
roleRef:
  kind: ClusterRole
  name: spire-server-role
  apiGroup: rbac.authorization.k8s.io
