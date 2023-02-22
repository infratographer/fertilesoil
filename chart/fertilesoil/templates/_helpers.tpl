{{/*
Template ingress hostname
*/}}
{{- define "ingress.hostname.rendered" -}}
{{- printf "%s" (tpl .Values.ingress.host $) -}}
{{- end -}}