---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "passthrough"
      networkservicemesh.io/impl: "secure-intranet-connectivity"
  replicas: 1
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "passthrough"
        networkservicemesh.io/impl: "secure-intranet-connectivity"
    spec:
      serviceAccount: nse-acc
      containers:
        - name: passthrough-nse
          image: {{ .Values.registry }}/{{ .Values.org }}/vpp-test-common:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: TEST_APPLICATION
              value: "vppagent-firewall-nse"
            - name: ENDPOINT_NETWORK_SERVICE
              value: "secure-intranet-connectivity"
            - name: ADVERTISE_NSE_LABELS
              value: "app=passthrough-1"
            - name: OUTGOING_NSC_NAME
              value: "secure-intranet-connectivity"
            - name: OUTGOING_NSC_LABELS
              value: "app=passthrough-1"
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
  name: vppagent-passthrough-nse-1
  namespace: {{ .Release.Namespace }}
---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "passthrough"
      networkservicemesh.io/impl: "secure-intranet-connectivity"
  replicas: 1
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "passthrough"
        networkservicemesh.io/impl: "secure-intranet-connectivity"
    spec:
      serviceAccount: nse-acc
      containers:
        - name: passthrough-nse
          image: {{ .Values.registry }}/{{ .Values.org }}/vpp-test-common:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: TEST_APPLICATION
              value: "vppagent-firewall-nse"
            - name: ENDPOINT_NETWORK_SERVICE
              value: "secure-intranet-connectivity"
            - name: ADVERTISE_NSE_LABELS
              value: "app=passthrough-2"
            - name: OUTGOING_NSC_NAME
              value: "secure-intranet-connectivity"
            - name: OUTGOING_NSC_LABELS
              value: "app=passthrough-2"
            - name: TRACER_ENABLED
              value: "true"
          resources:
            limits:
              networkservicemesh.io/socket: 1
metadata:
  name: vppagent-passthrough-nse-2
  namespace: {{ .Release.Namespace }}
---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "passthrough"
      networkservicemesh.io/impl: "secure-intranet-connectivity"
  replicas: 1
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "passthrough"
        networkservicemesh.io/impl: "secure-intranet-connectivity"
    spec:
      serviceAccount: nse-acc
      containers:
        - name: passthrough-nse
          image: {{ .Values.registry }}/{{ .Values.org }}/vpp-test-common:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: TEST_APPLICATION
              value: "vppagent-firewall-nse"
            - name: ENDPOINT_NETWORK_SERVICE
              value: "secure-intranet-connectivity"
            - name: ADVERTISE_NSE_LABELS
              value: "app=passthrough-3"
            - name: OUTGOING_NSC_NAME
              value: "secure-intranet-connectivity"
            - name: OUTGOING_NSC_LABELS
              value: "app=passthrough-3"
            - name: TRACER_ENABLED
              value: "true"
          resources:
            limits:
              networkservicemesh.io/socket: 1
metadata:
  name: vppagent-passthrough-nse-3
  namespace: {{ .Release.Namespace }}
