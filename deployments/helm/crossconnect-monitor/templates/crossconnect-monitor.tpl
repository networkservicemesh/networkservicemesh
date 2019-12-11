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
      containers:
        - name: crossconnect-monitor
          image: {{ .Values.registry }}/{{ .Values.org }}/crossconnect-monitor:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: INSECURE
{{- if .Values.insecure }}
              value: "true"
{{- else }}
              value: "false"
{{- end }}
          volumeMounts:
            - name: spire-agent-socket
              mountPath: /run/spire/sockets
              readOnly: true
      volumes:
        - hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
          name: spire-agent-socket
metadata:
  name: crossconnect-monitor
  namespace: {{ .Release.Namespace }}
