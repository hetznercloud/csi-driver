{{- /*
Expand the name of the chart.
*/}}
{{- define "hetzner.common.names.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- /*
Create chart name and version as used by the chart label.
*/}}
{{- define "hetzner.common.names.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- /*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "hetzner.common.names.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- /*
Allow the release namespace to be overridden for multi-namespace deployments in combined charts.
*/}}
{{- define "hetzner.common.names.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- /*
Create a fully qualified app name adding the installation's namespace.
*/}}
{{- define "hetzner.common.names.fullname.namespace" -}}
{{- printf "%s-%s" (include "hetzner.common.names.fullname" .) (include "hetzner.common.names.namespace" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- /*
Create the name of the service account to use
*/}}
{{- define "hetzner.common.names.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "hetzner.common.names.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}
