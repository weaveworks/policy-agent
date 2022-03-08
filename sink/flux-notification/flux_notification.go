package flux_notification

import (
	"context"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	mglx_events "github.com/MagalixCorp/magalix-policy-agent/pkg/events"
	"github.com/MagalixTechnologies/core/logger"
	"github.com/fluxcd/pkg/runtime/events"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	reportingController string = "magalix-policy-agent"
	resultChanSize      int    = 50
)

type FluxNotificationSink struct {
	recorder     record.EventRecorder
	resultChan   chan domain.PolicyValidation
	cancelWorker context.CancelFunc
	accountID    string
	clusterID    string
}

// NewFluxNotificationSink returns a sink that sends results to k8s events queue and flux notification controller
func NewFluxNotificationSink(mgr ctrl.Manager, webhook, accountID, clusterID string) (*FluxNotificationSink, error) {
	recorder, err := events.NewRecorder(mgr, mgr.GetLogger(), webhook, reportingController)
	if err != nil {
		return nil, err
	}

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
			logger.Info("received validation result")
			f.write(result)
		case <-ctx.Done():
			break
		}
	}
}

func (f *FluxNotificationSink) write(result domain.PolicyValidation) {
	fluxObject := getFluxObject(result.Entity.Labels)
	if fluxObject == nil {
		logger.Infow("ignoring result for resource")
		return
	}

	event := mglx_events.FromPolicyValidationResult(result)
	f.recorder.AnnotatedEventf(
		fluxObject,
		event.Annotations,
		event.Type,
		event.Reason,
		event.Message,
	)
}
