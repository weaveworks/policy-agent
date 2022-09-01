{{- define "isAuditFileSystemSinkEnabled" -}}
{{- if .Values.config.audit }}
{{- if eq .Values.config.audit.enabled true }}
{{- if .Values.config.audit.sinks }}
{{- if .Values.config.audit.sinks.fileSystemSink }}
{{- if .Values.config.audit.sinks.fileSystemSink.filePath }}
{{- printf "true"}}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end -}}

{{- define "isAdmissionFileSystemSinkEnabled" -}}
{{- if .Values.config.admission }}
{{- if eq .Values.config.admission.enabled true }}
{{- if .Values.config.admission.sinks }}
{{- if .Values.config.admission.sinks.fileSystemSink }}
{{- if .Values.config.admission.sinks.fileSystemSink.filePath }}
{{- printf "true"}}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end -}}
