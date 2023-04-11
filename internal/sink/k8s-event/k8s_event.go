package k8s_event

import (
	"context"
	"os"

	"github.com/weaveworks/policy-agent/internal/utils"
	"github.com/weaveworks/policy-agent/pkg/logger"
	"github.com/weaveworks/policy-agent/pkg/policy-core/domain"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
)

const (
	resultChanSize int = 50
)

type K8sEventSink struct {
	kubeClient          kubernetes.Interface
	resultChan          chan domain.PolicyValidation
	cancelWorker        context.CancelFunc
	accountID           string
	clusterID           string
	reportingController string
	reportingInstance   string
}

// NewK8sEventSink returns a sink that sends results to kubernetes events queue
func NewK8sEventSink(kubeClient kubernetes.Interface, accountID, clusterID, reportingController string) (*K8sEventSink, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	return &K8sEventSink{
		kubeClient:          kubeClient,
		resultChan:          make(chan domain.PolicyValidation, resultChanSize),
		accountID:           accountID,
		clusterID:           clusterID,
		reportingController: reportingController,
		reportingInstance:   hostname,
	}, nil
}

// Start starts the writer worker
func (k *K8sEventSink) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	k.cancelWorker = cancel
	return k.writeWorker(ctx)
}

// Stop stops worker
func (k *K8sEventSink) Stop() {
	k.cancelWorker()
}

// Write adds results to buffer, implements github.com/weaveworks/policy-agent/pkg/policy-core/domain.PolicyValidationSink
func (k *K8sEventSink) Write(_ context.Context, results []domain.PolicyValidation) error {
	logger.Infow("writing validation results", "sink", "k8s_events", "count", len(results))
	for _, result := range results {
		k.resultChan <- result
	}
	return nil
}

func (f *K8sEventSink) writeWorker(ctx context.Context) error {
	for {
		select {
		case result := <-f.resultChan:
			f.write(ctx, result)
		case <-ctx.Done():
			logger.Info("stopping write worker ...")
			return nil
		}
	}
}

func (k *K8sEventSink) write(ctx context.Context, result domain.PolicyValidation) {
	event, err := domain.NewK8sEventFromPolicyValidation(result)
	if err != nil {
		logger.Errorw(
			"failed to create event from policy validation",
			"error",
			err,
			"entity_kind", result.Entity.Kind,
			"entity_name", result.Entity.Name,
			"entity_namespace", result.Entity.Namespace,
			"policy", result.Policy.ID,
		)
		return
	}

	fluxObject := utils.GetFluxObject(result.Entity.Labels)
	if fluxObject != nil {
		event.InvolvedObject = v1.ObjectReference{
			UID:             fluxObject.GetUID(),
			APIVersion:      fluxObject.GetAPIVersion(),
			Kind:            fluxObject.GetKind(),
			Name:            fluxObject.GetName(),
			Namespace:       fluxObject.GetNamespace(),
			ResourceVersion: fluxObject.GetResourceVersion(),
		}
		event.Namespace = fluxObject.GetNamespace()
	}

	event.ReportingController = k.reportingController
	event.ReportingInstance = k.reportingInstance
	event.Source = v1.EventSource{Component: k.reportingController}

	logger.Debugw(
		"sending event ...",
		"type", event.Type,
		"entity_kind", result.Entity.Kind,
		"entity_name", result.Entity.Name,
		"entity_namespace", result.Entity.Namespace,
		"policy", result.Policy.ID,
	)

	_, err = k.kubeClient.CoreV1().Events(event.Namespace).Create(ctx, event, metav1.CreateOptions{})
	if err != nil {
		logger.Errorw("failed to send event", "error", err)
	}
}
