{{ $td := .Values.trustDomain }}

apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-bundle
  namespace: {{ .Values.namespace }}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-server
  namespace: {{ .Values.namespace }}
data:
  server.conf: |
    server {
      bind_address = "0.0.0.0"
      bind_port = "8081"
      trust_domain = {{ $td | quote }}
      data_dir = "/run/spire/data"
      log_level = "DEBUG"
      svid_ttl = "1h"
      upstream_bundle = true
      registration_uds_path = "/run/spire/sockets/registration.sock"
      ca_subject = {
        Country = ["US"],
        Organization = ["SPIFFE"],
        CommonName = "",
      }

      experimental {
              bundle_endpoint_enabled = true
              bundle_endpoint_port = 8443

              federates_with {{ (index .Values $td).federatesDomain }} {
                      bundle_endpoint_address = {{ (index .Values $td).federatesDomain | quote }}
                      bundle_endpoint_port = 8443
                      bundle_endpoint_spiffe_id = "spiffe://{{ (index .Values $td).federatesDomain }}/spire/server"
              }
      }
    }
    plugins {
      DataStore "sql" {
        plugin_data {
          database_type = "sqlite3"
          connection_string = "file::memory:?cache=shared"
          # connection_string = "/run/spire/data/datastore.sqlite3"
        }
      }
      NodeAttestor "k8s_sat" {
        plugin_data {
          clusters = {
            # NOTE: Change this to your cluster name
            "kubernetes" = {
              use_token_review_api_validation = true
              service_account_whitelist = ["{{ .Values.namespace }}:spire-agent"]
            }
          }
        }
      }
      NodeResolver "noop" {
        plugin_data {}
      }
      # KeyManager "disk" {
      #  plugin_data {
      #    keys_path = "/run/spire/data/keys.json"
      #  }
      # }
      KeyManager "memory" {
        plugin_data {
        }
      }
      {{- if not .Values.selfSignedCA }}
      UpstreamCA "disk" {
        plugin_data {
          ttl = "12h"
          key_file_path = "/run/spire/secret/bootstrap.key"
          cert_file_path = "/run/spire/secret/bootstrap.crt"
        }
      }
      {{- end }}
      Notifier "k8sbundle" {
        plugin_data {
          # This plugin updates the bundle.crt value in the spire:spire-bundle
          # ConfigMap by default, so no additional configuration is necessary.
        }
      }
    }
