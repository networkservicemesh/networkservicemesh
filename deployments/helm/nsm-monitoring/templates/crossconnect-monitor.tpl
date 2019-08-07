---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      app: nsmgr-daemonset
  template:
    metadata:
      labels:
        app: nsmgr-daemonset
    spec:
      serviceAccount: nsmgr-acc
      volumes:
        - hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
          name: spire-agent-socket
      containers:
        - name: crossconnect-monitor
          image: {{ .Values.registry }}/{{ .Values.org }}/crossconnect-monitor:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          volumeMounts:
            - name: spire-agent-socket
              mountPath: /run/spire/sockets
              readOnly: true
metadata:
  name: crossconnect-monitor
  namespace: {{ .Release.Namespace }}
