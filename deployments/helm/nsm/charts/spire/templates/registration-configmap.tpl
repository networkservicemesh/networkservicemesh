apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-entries
  namespace: {{ .Values.namespace }}
data:
  registration.json: |
    {
      "entries": [
        {
          "selectors": [
            {
              "type": "k8s_sat",
              "value": "agent_sa:spire-agent"
            }
          ],
          "spiffe_id": "spiffe://{{ .Values.trustDomain }}/spire-agent",
          "parent_id": "spiffe://{{ .Values.trustDomain }}/spire/server"
        },
        {
          "selectors": [
            {
              "type": "k8s",
              "value": "sa:nsmgr-acc"
            }
          ],
          "spiffe_id": "spiffe://{{ .Values.trustDomain }}/nsmgr",
          "parent_id": "spiffe://{{ .Values.trustDomain }}/spire-agent"
        },
        {
          "selectors": [
            {
              "type": "k8s",
              "value": "sa:nse-acc"
            }
          ],
          "spiffe_id": "spiffe://{{ .Values.trustDomain }}/nse",
          "parent_id": "spiffe://{{ .Values.trustDomain }}/spire-agent"
        },
        {
          "selectors": [
            {
              "type": "k8s",
              "value": "sa:nsc-acc"
            }
          ],
          "spiffe_id": "spiffe://{{ .Values.trustDomain }}/nsc",
          "parent_id": "spiffe://{{ .Values.trustDomain }}/spire-agent"
        },
        {
          "selectors": [
            {
              "type": "k8s",
              "value": "sa:forward-plane-acc"
            }
          ],
          "spiffe_id": "spiffe://{{ .Values.trustDomain }}/forward-plane",
          "parent_id": "spiffe://{{ .Values.trustDomain }}/spire-agent"
        }
      ]
    }



