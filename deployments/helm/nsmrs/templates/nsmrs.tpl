---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nsmrs
  namespace: {{ .Release.Namespace }}
spec:
  selector:
    matchLabels:
      run: nsmrs
  replicas: 1
  template:
    metadata:
      labels:
        run: nsmrs
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
      nodeSelector:
        nsmrs: "true"
