{{- define "kerbernetes.name" -}}
kerbernetes-api
{{- end }}

{{- define "kerbernetes.fullname" -}}
{{ .Release.Name }}-{{ include "kerbernetes.name" . }}
{{- end }}
