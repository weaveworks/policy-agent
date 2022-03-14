package auditor

import "context"

type AuditEventType string

const (
	AuditEventTypeInitial    AuditEventType = "initial-audit"
	AuditEventTypePeriodical AuditEventType = "periodic-audit"
	entitiesSizeLimit                       = 50
	TypeAudit                               = "Audit"
)

type AuditEvent struct {
	Type AuditEventType
	Data interface{}
}

type AuditEventListener func(ctx context.Context, auditEvent AuditEvent)
