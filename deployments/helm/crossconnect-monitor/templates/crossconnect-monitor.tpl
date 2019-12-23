---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      app: crossconnect-monitor
  template:
    metadata:
      labels:
        app: crossconnect-monitor
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
            - name: METRICS_COLLECTOR_ENABLED
{{- if .Values.metricsCollectorEnabled }}
              value: "true"
{{- else }}
              value: "false"
{{- end }}
            - name: PROMETHEUS
{{- if .Values.prometheus }}
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

{{- if .Values.prometheus }}
---
apiVersion: v1
kind: Service
metadata:
  name: crossconnect-monitor-svc
  namespace: nsm-system
  labels:
    app: crossconnect-monitor

spec:
  selector:
    app: crossconnect-monitor
  ports:
    - port: 9095
      protocol: TCP
      targetPort: 9090
{{- end }}

