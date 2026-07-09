{{/* Expand the name of the chart. */}}
{{- define "vmprober.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Fully qualified app name. */}}
{{- define "vmprober.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "vmprober.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/* Container image, defaulting the tag to "<appVersion>-alpine". */}}
{{- define "vmprober.image" -}}
{{- $tag := .Values.image.tag | default (printf "%s-alpine" .Chart.AppVersion) -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end }}

{{- define "vmprober.labels" -}}
helm.sh/chart: {{ include "vmprober.chart" . }}
{{ include "vmprober.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "vmprober.selectorLabels" -}}
app.kubernetes.io/name: {{ include "vmprober.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "vmprober.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "vmprober.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
