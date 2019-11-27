apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Chart.Name }}
spec:
  selector:
    matchLabels:
      app:  {{ .Chart.Name }}
  template:
    metadata:
      labels:
        app: {{ .Chart.Name }}
    spec:
      serviceAccountName: {{ .Values.serviceAccount.name }}
      containers:
        - name: {{ .Chart.Name }}
          image: {{ .Values.registry }}/{{ .Values.org }}/{{ .Chart.Name }}:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
