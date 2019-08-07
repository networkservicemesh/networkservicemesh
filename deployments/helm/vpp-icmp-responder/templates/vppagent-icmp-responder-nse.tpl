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
      volumes:
        - hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
          name: spire-agent-socket
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
            - name: ADVERTISE_NSE_NAME
              value: "icmp-responder"
            - name: ADVERTISE_NSE_LABELS
              value: "app=icmp-responder"
            - name: TRACER_ENABLED
              value: "true"
            - name: IP_ADDRESS
              value: "10.30.1.0/24"
          volumeMounts:
            - name: spire-agent-socket
              mountPath: /run/spire/sockets
              readOnly: true
          resources:
            limits:
              networkservicemesh.io/socket: 1
metadata:
  name: vppagent-icmp-responder-nse
  namespace: {{ .Release.Namespace }}
