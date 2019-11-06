---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nsmrs
  namespace: {{ .Release.Namespace }}
spec:
  selector:
    matchLabels:
      app: nsmrs-daemonset
  template:
    metadata:
      labels:
        app: nsmrs-daemonset
    spec:
      serviceAccount: nsmrs-acc
      containers:
        - name: nsmrs
          image: {{ .Values.registry }}/{{ .Values.org }}/nsmrs:{{ .Values.tag }}
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
          ports:
            - containerPort: 5010
              hostPort: 80
      volumes:
        - hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
          name: spire-agent-socket
