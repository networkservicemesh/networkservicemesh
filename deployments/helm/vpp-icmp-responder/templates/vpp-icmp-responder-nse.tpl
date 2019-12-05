---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "icmp-responder"
      networkservicemesh.io/impl: "vppagent-icmp-responder"
  replicas: 2
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "icmp-responder"
        networkservicemesh.io/impl: "vppagent-icmp-responder"
    spec:
      serviceAccount: nse-acc
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: networkservicemesh.io/app
                    operator: In
                    values:
                      - icmp-responder
                  - key: networkservicemesh.io/impl
                    operator: In
                    values:
                      - vppagent-icmp-responder
              topologyKey: "kubernetes.io/hostname"
      containers:
        - name: icmp-responder-nse
          image: {{ .Values.registry }}/{{ .Values.org }}/vpp-test-common:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: TEST_APPLICATION
              value: "vppagent-icmp-responder-nse"
            - name: ENDPOINT_NETWORK_SERVICE
              value: "icmp-responder"
            - name: ADVERTISE_NSE_LABELS
              value: "app=icmp-responder"
            - name: IP_ADDRESS
              value: "10.30.1.0/24"
{{- if .Values.global.JaegerTracing }}
            - name: TRACER_ENABLED
              value: "true"
            - name: JAEGER_AGENT_HOST
              value: jaeger.nsm-system
            - name: JAEGER_AGENT_PORT
              value: "6831"
{{- end }}
          resources:
            limits:
              networkservicemesh.io/socket: 1
metadata:
  name: vpp-icmp-responder-nse
  namespace: {{ .Release.Namespace }}
