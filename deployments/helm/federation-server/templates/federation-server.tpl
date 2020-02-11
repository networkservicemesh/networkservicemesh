---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: federation-server
  namespace: {{ .Release.Namespace }}
spec:
  selector:
    matchLabels:
      app: federation-server
  replicas: 1
  template:
    metadata:
      labels:
        app: federation-server
    spec:
      serviceAccount: federation-server-acc
      containers:
        - name: federation-server
          image: {{ .Values.registry }}/{{ .Values.org }}/federation-server:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          ports:
            - containerPort: 7002
              hostPort: 7002

---
apiVersion: v1
kind: Service
metadata:
  name: federation-server
  labels:
    app: federation-server
  namespace: {{ .Release.Namespace }}
spec:
  clusterIP: None
  ports:
    - name: register
      port: 7002
      protocol: TCP
  selector:
    app: federation-server