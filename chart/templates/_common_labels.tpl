{{- /*
Kubernetes standard labels
*/}}
{{- define "hetzner.common.labels.standard" -}}
app.kubernetes.io/name: {{ include "hetzner.common.names.name" . }}
helm.sh/chart: {{ include "hetzner.common.names.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- /*
Labels to use on deploy.spec.selector.matchLabels and svc.spec.selector
*/}}
{{- define "hetzner.common.labels.matchLabels" -}}
app.kubernetes.io/name: {{ include "hetzner.common.names.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
