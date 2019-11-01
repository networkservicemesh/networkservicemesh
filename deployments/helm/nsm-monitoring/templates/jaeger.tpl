apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
  namespace: {{ .Release.Namespace }}
spec:
  selector:
    matchLabels:
      run: jaeger
  replicas: 1
  template:
    metadata:
      labels:
        run: jaeger
    spec:
      containers:
        - name: jaeger
          image: {{ .Values.image }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          ports:
            - name: http
              containerPort: 16686
            - name: jaeger
              containerPort: 6831
              protocol: UDP
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger
  namespace: {{ .Release.Namespace }}
  labels:
    run: jaeger
  namespace: {{ .Release.Namespace }}
spec:
  ports:
    - name: http
{{- if eq .Values.monSvcType "NodePort" }}
      nodePort: 31922
{{- end }}
      port: 16686
      protocol: TCP
    - name: jaeger
      port: 6831
      protocol: UDP
  selector:
    run: jaeger
  type: {{ .Values.monSvcType }}
