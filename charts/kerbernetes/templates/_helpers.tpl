{{/* Define the full app name including release */}}
{{- define "kerbernetes-api.fullname" -}}
{{ .Release.Name }}-kerbernetes-api
{{- end }}

{{/* Define the app label (same as fullname) */}}
{{- define "kerbernetes-api.appLabel" -}}
app: {{ include "kerbernetes-api.fullname" . }}
{{- end }}

{{/* Define the container name */}}
{{- define "kerbernetes-api.containerName" -}}
kerbernetes-api
{{- end }}
