apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: nsmgr
  namespace: {{ .Release.Namespace }}
spec:
  selector:
    matchLabels:
      app: nsmmgr-daemonset
  template:
    metadata:
      labels:
        app: nsmmgr-daemonset
    spec:
      containers:
        - name: nsmdp
          image: {{ .Values.registry }}/networkservicemesh/nsmdp:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          volumeMounts:
            - name: kubelet-socket
              mountPath: /var/lib/kubelet/device-plugins
            - name: nsm-socket
              mountPath: /var/lib/networkservicemesh
        - name: nsmd
          image: {{ .Values.registry }}/networkservicemesh/nsmd:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          volumeMounts:
            - name: nsm-socket
              mountPath: /var/lib/networkservicemesh
            - name: nsm-plugin-socket
              mountPath: /var/lib/networkservicemesh/plugins
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
        - name: nsmd-k8s
          image: {{ .Values.registry }}/networkservicemesh/nsmd-k8s:{{ .Values.tag }}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
      volumes:
        - hostPath:
            path: /var/lib/kubelet/device-plugins
            type: DirectoryOrCreate
          name: kubelet-socket
        - hostPath:
            path: /var/lib/networkservicemesh
            type: DirectoryOrCreate
          name: nsm-socket
        - hostPath:
            path: /var/lib/networkservicemesh/plugins
            type: DirectoryOrCreate
          name: nsm-plugin-socket