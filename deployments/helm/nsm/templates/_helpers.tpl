---
{{/* Configure Jaeger environment */}}
{{- define "jaeger.env" -}}
{{- if .Values.global.JaegerTracing }}
- name: TRACER_ENABLED
  value: "true"
- name: JAEGER_AGENT_HOST
  value: "jaeger.nsm-system"
- name: JAEGER_AGENT_PORT
  value: "6831"
{{- end }}
{{- end -}}

{{/* Configure INSECURE environment variable */}}
{{- define "insecure.env" }}
- name: INSECURE
  value: {{ .Values.insecure | default false | quote }}
{{- end -}}
