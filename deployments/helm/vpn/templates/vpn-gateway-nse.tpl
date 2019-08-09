---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "vpn-gateway"
      networkservicemesh.io/impl: "secure-intranet-connectivity"
  replicas: 1
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "vpn-gateway"
        networkservicemesh.io/impl: "secure-intranet-connectivity"
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: networkservicemesh.io/app
                    operator: In
                    values:
                      - vpn-gateway
              topologyKey: "kubernetes.io/hostname"
      containers:
        - name: vpn-gateway
          image: {{ .Values.registry }}/{{ .Values.org }}/test-common:{{ .Values.tag }}
          command: ["/bin/icmp-responder-nse"]
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: ADVERTISE_NSE_NAME
              value: "secure-intranet-connectivity"
            - name: ADVERTISE_NSE_LABELS
              value: "app=vpn-gateway"
            - name: TRACER_ENABLED
              value: "true"
            - name: IP_ADDRESS
              value: "10.60.1.0/24"
          resources:
            limits:
              networkservicemesh.io/socket: 1
        - name: nginx
          image: {{ .Values.registry }}/{{ .Values.org }}/nginx:{{ .Values.tag }}
metadata:
  name: vpn-gateway-nse
  namespace: {{ .Release.Namespace }}
