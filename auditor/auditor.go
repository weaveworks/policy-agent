package auditor

import (
	"context"
	"time"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/validation"
	"github.com/MagalixTechnologies/core/logger"
)

// AuditorController performs audit on regular interval by uing entitites sources to retrieve resources
type AuditorController struct {
	entitiesSources []domain.EntitiesSource
	auditEvent      chan AuditEvent
	validator       validation.Validator
}

const (
	auditInterval = 23 * time.Hour
)

// NewAuditController returns a new instance of AuditController
func NewAuditController(validator validation.Validator, entitiesSources ...domain.EntitiesSource) *AuditorController {
	return &AuditorController{
		entitiesSources: entitiesSources,
		auditEvent:      make(chan AuditEvent, 1),
		validator:       validator,
	}
}

// Run starts the audit controller
func (a *AuditorController) Run(ctx context.Context) error {
	cancelCtx, _ := context.WithCancel(ctx)
	auditTicker := time.NewTicker(auditInterval)
	defer auditTicker.Stop()

	a.Audit(AuditEventTypeInitial, nil)
	for {
		select {
		case <-cancelCtx.Done():
			return nil
		case <-auditTicker.C:
			a.doAudit(AuditEventTypePeriodical)
		case event := <-a.auditEvent:
			a.doAudit(event.Type)
		}
	}
}

func (a *AuditorController) doAudit(auditType AuditEventType) {
	logger.Infof("starting %s", auditType)
	for i := range a.entitiesSources {
		hasNext := true
		keySet := ""
		for hasNext {
			entitySource := a.entitiesSources[i]
			opts := domain.ListOptions{
				Limit:  entitiesSizeLimit,
				KeySet: keySet,
			}
			ctx := context.Background()
			entitiesList, err := entitySource.List(ctx, &opts)
			if err != nil {
				logger.Errorw("failed to list entities during audit", "kind", entitySource.Kind(), "error", err)
				continue
			}
			hasNext = entitiesList.HasNext
			keySet = entitiesList.KeySet

			for idx := range entitiesList.Data {
				entity := entitiesList.Data[idx]
				_, err := a.validator.Validate(ctx, entity, TypeAudit, string(auditType))
				if err != nil {
					logger.Errorw(
						"failed to validate entity during audit",
						"entity-kind", entity.Kind,
						"entity-name", entity.Name,
						"error", err)
				}
			}
		}
	}
	logger.Info("finished audit")
}

// Audit triggers an audit with specified audit type
func (a *AuditorController) Audit(auditType AuditEventType, data interface{}) {
	a.auditEvent <- AuditEvent{
		Type: auditType,
		Data: data,
	}
}
