{{- define "otel-stack.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "otel-stack.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := include "otel-stack.name" . -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "otel-stack.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "otel-stack.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "otel-stack.selectorLabelsBackend" -}}
app.kubernetes.io/name: {{ include "otel-stack.name" . }}-backend
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "otel-stack.selectorLabelsUI" -}}
app.kubernetes.io/name: {{ include "otel-stack.name" . }}-ui
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}