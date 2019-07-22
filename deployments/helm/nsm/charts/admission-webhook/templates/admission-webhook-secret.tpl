{{- $ca := genCA "admission-controller-ca" 3650 -}}
{{- $cn := printf "nsm-admission-webhook-svc" -}}
{{- $altName1 := printf "%s.%s" $cn .Release.Namespace }}
{{- $altName2 := printf "%s.%s.svc" $cn .Release.Namespace }}
{{- $cert := genSignedCert $cn nil (list $altName1 $altName2) 3650 $ca -}}

apiVersion: v1
kind: Secret
metadata:
  name: nsm-admission-webhook-certs
  namespace: {{ .Release.Namespace }}
type: Opaque
data:
  key.pem: {{ $cert.Key | b64enc }}
  cert.pem: {{ $cert.Cert | b64enc }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nsm-admission-webhook
  namespace: {{ .Release.Namespace }}
  labels:
    app: nsm-admission-webhook
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nsm-admission-webhook
  template:
    metadata:
      labels:
        app: nsm-admission-webhook
    spec:
      containers:
        - name: nsm-admission-webhook
          image: {{ .Values.registry }}/{{ .Values.org }}/admission-webhook:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: REPO
              value: "{{ .Values.org }}"
            - name: TAG
              value: "{{ .Values.tag }}"
{{- if .Values.global.JaegerTracing }}
          env:
            - name: JAEGER_SERVICE_HOST
              value: jaeger.nsm-system
            - name: JAEGER_SERVICE_PORT_JAEGER
              value: "6831"
{{- end }}
          volumeMounts:
            - name: webhook-certs
              mountPath: /etc/webhook/certs
              readOnly: true
      volumes:
        - name: webhook-certs
          secret:
            secretName: nsm-admission-webhook-certs
---
apiVersion: v1
kind: Service
metadata:
  name: nsm-admission-webhook-svc
  namespace: {{ .Release.Namespace }}
  labels:
    app: nsm-admission-webhook
spec:
  ports:
    - port: 443
      targetPort: 443
  selector:
    app: nsm-admission-webhook
---
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingWebhookConfiguration
metadata:
  name: nsm-admission-webhook-cfg
  namespace: {{ .Release.Namespace }}
  labels:
    app: nsm-admission-webhook
webhooks:
  - name: admission-webhook.networkservicemesh.io
    clientConfig:
      service:
        name: nsm-admission-webhook-svc
        namespace: {{ .Release.Namespace }}
        path: "/mutate"
      caBundle: {{ $ca.Cert | b64enc }}
    rules:
      - operations: ["CREATE"]
        apiGroups: ["apps", "extensions", ""]
        apiVersions: ["v1", "v1beta1"]
        resources: ["deployments", "services", "pods"]
