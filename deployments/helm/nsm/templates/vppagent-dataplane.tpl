---
apiVersion: apps/v1
kind: DaemonSet
spec:
  selector:
    matchLabels:
      app: nsm-vpp-dataplane
  template:
    metadata:
      labels:
        app: nsm-vpp-dataplane
    spec:
      hostPID: true
      hostNetwork: true
      containers:
        - name: vppagent-dataplane
          securityContext:
            privileged: true
          image: {{ .Values.registry }}/{{ .Values.org }}/vppagent-dataplane:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: NSM_DATAPLANE_SRC_IP
              valueFrom:
                fieldRef:
                  fieldPath: status.podIP
          volumeMounts:
            - name: workspace
              mountPath: /var/lib/networkservicemesh/
              mountPropagation: Bidirectional
          livenessProbe:
            httpGet:
              path: /liveness
              port: 5555
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 3
          readinessProbe:
            httpGet:
              path: /readiness
              port: 5555
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 3
      volumes:
        - hostPath:
            path: /var/lib/networkservicemesh
            type: DirectoryOrCreate
          name: workspace
metadata:
  name: nsm-vppagent-dataplane
  namespace: {{ .Release.Namespace }}
