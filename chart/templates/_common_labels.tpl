{{- /*
Kubernetes standard labels
*/}}
{{- define "hcloud-csi.labels.standard" -}}
app.kubernetes.io/name: {{ include "hcloud-csi.names.name" . }}
helm.sh/chart: {{ include "hcloud-csi.names.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- /*
Labels to use on deploy.spec.selector.matchLabels and svc.spec.selector
*/}}
{{- define "hcloud-csi.labels.matchLabels" -}}
app.kubernetes.io/name: {{ include "hcloud-csi.names.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
