{{/*
Return the Container Image Name
{{ include "common.images.image" (dict "value" .Values.controller.image.hcloudCSIDriver "context" .) }}
*/}}
{{- define "common.images.image" -}}
{{ tpl .value.name .context }}{{ if .value.tag }}:{{ tpl .value.tag .context }}{{ end }}
{{- end -}}

{{/*
Return the proper Container Image Registry Secret Names evaluating values as templates
{{ include "common.images.pullSecrets" ( dict "images" (list .Values.path.to.the.image1 .Values.path.to.the.image2) "context" $) }}
*/}}
{{- define "common.images.pullSecrets" -}}
  {{- $pullSecrets := list }}
  {{- $context := .context }}

  {{- if $context.Values.global }}
    {{- range $context.Values.global.imagePullSecrets -}}
      {{- $pullSecrets = append $pullSecrets (include "common.tplvalues.render" (dict "value" . "context" $context)) -}}
    {{- end -}}
  {{- end -}}

  {{- range .images -}}
    {{- range .pullSecrets -}}
      {{- $pullSecrets = append $pullSecrets (include "common.tplvalues.render" (dict "value" . "context" $context)) -}}
    {{- end -}}
  {{- end -}}

  {{- if (not (empty $pullSecrets)) }}
imagePullSecrets:
    {{- range $pullSecrets }}
  - name: {{ . }}
    {{- end }}
  {{- end }}
{{- end -}}