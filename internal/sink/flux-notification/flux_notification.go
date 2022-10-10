package flux_notification

import (
	"context"
	"fmt"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/weaveworks/policy-agent/internal/utils"
	"k8s.io/client-go/tools/record"
)

const (
	resultChanSize int = 50
)

type FluxNotificationSink struct {
	recorder     record.EventRecorder
	resultChan   chan domain.PolicyValidation
	cancelWorker context.CancelFunc
	accountID    string
	clusterID    string
}

// NewFluxNotificationSink returns a sink that sends results to flux notification controller
func NewFluxNotificationSink(recorder record.EventRecorder, webhook, accountID, clusterID string) (*FluxNotificationSink, error) {
	return &FluxNotificationSink{
		recorder:   recorder,
		resultChan: make(chan domain.PolicyValidation, resultChanSize),
		accountID:  accountID,
		clusterID:  clusterID,
	}, nil
}

// Start starts the writer worker
func (f *FluxNotificationSink) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	f.cancelWorker = cancel
	return f.writeWorker(ctx)
}

// Stop stops worker
func (f *FluxNotificationSink) Stop() {
	f.cancelWorker()
}

// Write adds results to buffer, implements github.com/MagalixTechnologies/policy-core/domain.PolicyValidationSink
func (f *FluxNotificationSink) Write(_ context.Context, results []domain.PolicyValidation) error {
	logger.Debugw("writing validation results", "sink", "flux_notification", "count", len(results))
	for _, result := range results {
		f.resultChan <- result
	}
	return nil
}

func (f *FluxNotificationSink) writeWorker(ctx context.Context) error {
	for {
		select {
		case result := <-f.resultChan:
			f.write(result)
		case <-ctx.Done():
			logger.Info("stopping write worker ...")
			return nil
		}
	}
}

func (f *FluxNotificationSink) write(result domain.PolicyValidation) {
	fluxObject := utils.GetFluxObject(result.Entity.Labels)
	if fluxObject == nil {
		logger.Debugw(
			fmt.Sprintf("discarding %s result for orphan entity", result.Type),
			"kind", result.Entity.Kind,
			"name", result.Entity.Name,
			"namespace", result.Entity.Namespace,
		)
		return
	}

	event, err := domain.NewK8sEventFromPolicyValidation(result)
	if err != nil {
		logger.Errorw(
			"failed to create event from policy validation for flux notification",
			"error",
			err,
			"entity_kind", result.Entity.Kind,
			"entity_name", result.Entity.Name,
			"entity_namespace", result.Entity.Namespace,
			"policy", result.Policy.ID,
		)
		return
	}

	logger.Debugw(
		"sending event ...",
		"type", event.Type,
		"entity_kind", result.Entity.Kind,
		"entity_name", result.Entity.Name,
		"entity_namespace", result.Entity.Namespace,
		"policy", result.Policy.ID,
	)

	f.recorder.AnnotatedEventf(
		fluxObject,
		event.Annotations,
		event.Type,
		event.Reason,
		event.Message,
	)
}
