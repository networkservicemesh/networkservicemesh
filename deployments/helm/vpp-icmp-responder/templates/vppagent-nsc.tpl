---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io: "true"
      networkservicemesh.io/app: "vppagent-nsc"
  replicas: 2
  template:
    metadata:
      labels:
        networkservicemesh.io: "true"
        networkservicemesh.io/app: "vppagent-nsc"
    spec:
      hostPID: true
      serviceAccount: nsc-acc
      volumes:
        - hostPath:
            path: /run/spire/sockets
            type: DirectoryOrCreate
          name: spire-agent-socket
      containers:
        - name: vppagent-nsc
          image: {{ .Values.registry }}/{{ .Values.org }}/vpp-test-common:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: TEST_APPLICATION
              value: "vppagent-nsc"
            - name: OUTGOING_NSC_LABELS
              value: "app=icmp"
            - name: OUTGOING_NSC_NAME
              value: "icmp-responder"
          volumeMounts:
            - name: spire-agent-socket
              mountPath: /run/spire/sockets
              readOnly: true
          resources:
            limits:
              networkservicemesh.io/socket: 1
metadata:
  name: vppagent-nsc
namespace: {{ .Release.Namespace }}
