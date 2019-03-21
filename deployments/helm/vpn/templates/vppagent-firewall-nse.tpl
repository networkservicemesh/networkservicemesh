---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "firewall"
      networkservicemesh.io/impl: "secure-intranet-connectivity"
  replicas: 1
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "firewall"
        networkservicemesh.io/impl: "secure-intranet-connectivity"
    spec:
      containers:
        - name: firewall-nse
          image: {{ .Values.registry }}/{{ .Values.org }}/vpp-test-common:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: TEST_APPLICATION
              value: "vppagent-firewall-nse"
            - name: ENDPOINT_NETWORK_SERVICE
              value: "secure-intranet-connectivity"
            - name: ENDPOINT_LABELS
              value: "app=firewall"
            - name: CLIENT_NETWORK_SERVICE
              value: "secure-intranet-connectivity"
            - name: CLIENT_LABELS
              value: "app=firewall"
            - name: TRACER_ENABLED
              value: "true"
          resources:
            limits:
              networkservicemesh.io/socket: 1
          volumeMounts:
            - mountPath: /etc/vppagent-firewall/config.yaml
              subPath: config.yaml
              name: vppagent-firewall-config-volume
      volumes:
        - name: vppagent-firewall-config-volume
          configMap:
            name: vppagent-firewall-config-file
metadata:
  name: vppagent-firewall-nse
  namespace: {{ .Release.Namespace }}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: vppagent-firewall-config-file
  namespace: {{ .Release.Namespace }}
data:
  config.yaml: |
    aclRules:
      "Allow ICMP": "action=reflect,icmptype=8"
      "Allow TCP 80": "action=reflect,tcplowport=80,tcpupport=80"
