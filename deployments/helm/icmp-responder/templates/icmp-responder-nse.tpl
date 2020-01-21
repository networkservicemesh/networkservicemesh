---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "icmp-responder"
      networkservicemesh.io/impl: "icmp-responder"
  replicas: 1
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "icmp-responder"
        networkservicemesh.io/impl: "icmp-responder"
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
                      - icmp-responder
              topologyKey: "kubernetes.io/hostname"
      containers:
        - name: icmp-responder-nse
          image: {{ .Values.registry }}/{{ .Values.org }}/test-common:{{ .Values.tag}}
          command: ["/bin/icmp-responder-nse"]
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: NSM_SRIOV_RESOURCE_NAME
              value: "PCIDEVICE_KERNEL-SVC-1_INTEL_COM_10G" # TODO: fix this eyesore
            - name: ENDPOINT_NETWORK_SERVICE
              value: "icmp-responder"
            - name: ENDPOINT_LABELS
              value: "app=icmp-responder"
            - name: TRACER_ENABLED
              value: "true"
            - name: IP_ADDRESS
              value: "172.16.1.0/24"
            - name: NSM_NAMESPACE
              value: "nsm-system"
            - name: MECHANISM_TYPE
              value: SRIOV_KERNEL_INTERFACE
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
              kernel-svc-1.intel.com/10G: 1
              networkservicemesh.io/socket: 1
metadata:
  name: icmp-responder-nse
  namespace: {{ .Release.Namespace }}
