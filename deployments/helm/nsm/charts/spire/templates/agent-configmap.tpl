apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-agent
  namespace: {{ .Values.namespace }}
data:
  agent.conf: |
    agent {
      data_dir = "/run/spire"
      log_level = "DEBUG"
      server_address = "spire-server"
      server_port = "8081"
      socket_path = "/run/spire/sockets/agent.sock"
      trust_bundle_path = "/run/spire/bundle/bundle.crt"
      trust_domain = "test.com"
    }
    plugins {
      NodeAttestor "k8s_sat" {
        plugin_data {
          # NOTE: Change this to your cluster name
          cluster = "kubernetes"
        }
      }
      KeyManager "memory" {
        plugin_data {
        }
      }
      WorkloadAttestor "k8s" {
        plugin_data {
          {{- if .Values.azure }}
          kubelet_read_only_port = 10255
          {{- else }}
          skip_kubelet_verification = true
          {{- end }}
        }
      }
    }
