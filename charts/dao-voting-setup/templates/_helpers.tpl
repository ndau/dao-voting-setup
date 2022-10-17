{{- define "name" -}}
{{- default $.Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "serviceAccountName" -}}
{{- $name := include "name" . -}}
{{- default $name .Values.serviceAccountNameOverride -}}
{{- end -}}

{{- define "envVars" -}}
{{- if hasKey .Values.env .ContainerName }}
{{- $extraEnvs := index .Values.env .ContainerName }}
{{- $lenExtraEnvs := len $extraEnvs }}
{{- if gt $lenExtraEnvs 0 }}
env:
- name: NDAU_ENVIRONMENT
  value: {{ include "environmentName" . }}
{{ toYaml $extraEnvs }}
{{- end }}
{{- else if hasKey .Values.env "default" }}
{{- $extraEnvs := index .Values.env "default" }}
{{- $lenExtraEnvs := len $extraEnvs }}
{{- if gt $lenExtraEnvs 0 }}
env:
- name: NDAU_ENVIRONMENT
  value: {{ include "environmentName" . }}
{{ toYaml $extraEnvs }}
{{- end }}
{{- end }}
{{- end }}

{{- define "resources" -}}
{{- if hasKey .Values.resources .ContainerName }}
resources:
{{ toYaml (index .Values.resources .ContainerName) }}
{{- else if hasKey .Values.resources "default" }}
resources:
{{ toYaml .Values.resources.default | trim | indent 2 }}
{{- else }}
resources:
  limits:
    cpu: 1000m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
{{- end }}
{{- end }}
