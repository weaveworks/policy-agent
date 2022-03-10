package flux_notification

import (
	"context"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	mglx_events "github.com/MagalixCorp/magalix-policy-agent/pkg/events"
	"github.com/MagalixTechnologies/core/logger"
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
func (f *FluxNotificationSink) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	f.cancelWorker = cancel
	go f.writeWorker(ctx)
}

// Stop stops worker
func (f *FluxNotificationSink) Stop() {
	f.cancelWorker()
}

// Write adds results to buffer, implements github.com/MagalixCorp/magalix-policy-agent/pkg/domain.PolicyValidationSink
func (f *FluxNotificationSink) Write(_ context.Context, results []domain.PolicyValidation) error {
	logger.Infow("received validation results", "count", len(results))
	for _, result := range results {
		result.AccountID = f.accountID
		result.ClusterID = f.clusterID
		f.resultChan <- result
	}
	return nil
}

func (f *FluxNotificationSink) writeWorker(ctx context.Context) {
	for {
		select {
		case result := <-f.resultChan:
			f.write(result)
		case <-ctx.Done():
			logger.Info("stopping write worker ...")
			break
		}
	}
}

func (f *FluxNotificationSink) write(result domain.PolicyValidation) {
	fluxObject := getFluxObject(result.Entity.Labels)
	if fluxObject == nil {
		logger.Infow(
			"discarding result for orphan entity",
			"name", result.Entity.Name,
			"namespace", result.Entity.Namespace,
		)
		return
	}

	event := mglx_events.EventFromPolicyValidationResult(result)

	logger.Infow(
		"sending event ...",
		"type", event.Type,
		"resource_name", result.Entity.Name,
		"resource_namespace", result.Entity.Namespace,
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
