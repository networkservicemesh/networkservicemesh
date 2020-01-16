apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: spire-server
  namespace: {{ .Values.namespace }}
  labels:
    app: spire-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: spire-server
  serviceName: spire-server
  template:
    metadata:
      namespace: spire
      labels:
        app: spire-server
    spec:
      serviceAccountName: spire-server
      shareProcessNamespace: true
      containers:
        - name: nsm-spire
          securityContext:
            privileged: true
          image: {{ .Values.registry }}/{{ .Values.org }}/nsm-spire:{{ .Values.tag }}
          volumeMounts:
            - name: spire-server-socket
              mountPath: /run/spire/sockets
              readOnly: true
            - name: spire-entries
              mountPath: /run/spire/entries
              readOnly: true

        - name: spire-server
          image: gcr.io/spiffe-io/spire-server:0.9.0
          args:
            - -config
            - /run/spire/config/server.conf
          ports:
            - containerPort: 8081
          volumeMounts:
            - name: spire-server-socket
              mountPath: /run/spire/sockets
              readOnly: false
            - name: spire-config
              mountPath: /run/spire/config
              readOnly: true
            - name: spire-data
              mountPath: /run/spire/data
              readOnly: false
            - name: spire-secret
              mountPath: /run/spire/secret
          livenessProbe:
            tcpSocket:
              port: 8081
            failureThreshold: 2
            initialDelaySeconds: 15
            periodSeconds: 60
            timeoutSeconds: 3
      volumes:
        - name: spire-server-socket
          hostPath:
            path: /run/spire/server-sockets
            type: DirectoryOrCreate
        - name: spire-config
          configMap:
            name: spire-server
        - name: spire-secret
          secret:
            secretName: spire-secret
        - name: spire-entries
          configMap:
            name: spire-entries
        - name: spire-data
          hostPath:
            path: /var/spire-data
            type: DirectoryOrCreate
