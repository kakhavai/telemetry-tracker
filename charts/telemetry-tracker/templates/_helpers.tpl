{{- define "telemetry-tracker.name" -}}
telemetry-tracker
{{- end }}

{{- define "telemetry-tracker.fullname" -}}
{{ .Release.Name }}-{{ include "telemetry-tracker.name" . }}
{{- end }}
