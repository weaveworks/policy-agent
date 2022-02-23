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
	entitiesSources    []domain.EntitiesSource
	auditEvent         chan AuditEvent
	validator          validation.Validator
	auditEventListener AuditEventListener
	auditInterval      time.Duration
}

// NewAuditController returns a new instance of AuditController
func NewAuditController(validator validation.Validator, auditInterval time.Duration, entitiesSources ...domain.EntitiesSource) *AuditorController {
	auditController := &AuditorController{
		entitiesSources: entitiesSources,
		auditEvent:      make(chan AuditEvent, 1),
		validator:       validator,
		auditInterval:   auditInterval,
	}
	auditController.auditEventListener = auditController.doAudit
	return auditController
}

// RegisterAuditEventListener add a listener that reacts to audit events
func (a *AuditorController) RegisterAuditEventListener(auditEventListener AuditEventListener) {
	a.auditEventListener = auditEventListener
}

// Run starts the audit controller
func (a *AuditorController) Run(ctx context.Context) error {
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	auditTicker := time.NewTicker(a.auditInterval)
	defer auditTicker.Stop()
	defer cancelFunc()

	for {
		select {
		case <-cancelCtx.Done():
			return nil
		case <-auditTicker.C:
			auditEvent := AuditEvent{Type: AuditEventTypePeriodical}
			a.auditEventListener(ctx, auditEvent)
		case event := <-a.auditEvent:
			a.auditEventListener(ctx, event)
		}
	}
}

func (a *AuditorController) doAudit(ctx context.Context, auditEvent AuditEvent) {
	logger.Infof("starting %s", auditEvent.Type)
	for i := range a.entitiesSources {
		hasNext := true
		keySet := ""
		entitySource := a.entitiesSources[i]
		for hasNext {
			opts := domain.ListOptions{
				Limit:  entitiesSizeLimit,
				KeySet: keySet,
			}
			entitiesList, err := entitySource.List(ctx, &opts)
			if err != nil {
				logger.Errorw("failed to list entities during audit", "kind", entitySource.Kind(), "error", err)
				continue
			}
			hasNext = entitiesList.HasNext
			keySet = entitiesList.KeySet

			for idx := range entitiesList.Data {
				entity := entitiesList.Data[idx]
				_, err := a.validator.Validate(ctx, entity, TypeAudit, string(auditEvent.Type))
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
