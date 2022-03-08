package k8s_event

import (
	"context"
	"os"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	mglx_events "github.com/MagalixCorp/magalix-policy-agent/pkg/events"
	"github.com/MagalixTechnologies/core/logger"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
)

const (
	reportingController     = "magalix-policy-agent"
	resultChanSize      int = 50
)

type K8sEventSink struct {
	kubeClient        kubernetes.Interface
	resultChan        chan domain.PolicyValidation
	cancelWorker      context.CancelFunc
	accountID         string
	clusterID         string
	reportingInstance string
}

func NewK8sEventSink(kubeClient kubernetes.Interface, accountID, clusterID string) (*K8sEventSink, error) {
	hostname, err := os.Hostname()
	if err != nil {
		logger.Error(err, "failed to get hostname")
		return nil, err
	}

	return &K8sEventSink{
		kubeClient:        kubeClient,
		resultChan:        make(chan domain.PolicyValidation, resultChanSize),
		accountID:         accountID,
		clusterID:         clusterID,
		reportingInstance: hostname,
	}, nil
}

// Start starts the writer worker
func (k *K8sEventSink) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	k.cancelWorker = cancel
	go k.writeWorker(ctx)
}

// Stop stops worker
func (k *K8sEventSink) Stop() {
	k.cancelWorker()
}

// Write adds results to buffer, implements github.com/MagalixCorp/magalix-policy-agent/pkg/domain.PolicyValidationSink
func (k *K8sEventSink) Write(_ context.Context, results []domain.PolicyValidation) error {
	for _, result := range results {
		result.AccountID = k.accountID
		result.ClusterID = k.clusterID
		k.resultChan <- result
	}
	return nil
}

func (f *K8sEventSink) writeWorker(ctx context.Context) {
	for {
		select {
		case result := <-f.resultChan:
			f.write(ctx, result)
		case <-ctx.Done():
			break
		}
	}
}

func (k *K8sEventSink) write(ctx context.Context, result domain.PolicyValidation) {
	event := mglx_events.FromPolicyValidationResult(result)
	event.ReportingController = reportingController
	event.ReportingInstance = k.reportingInstance
	event.Source = v1.EventSource{Component: reportingController}

	_, err := k.kubeClient.CoreV1().Events(event.Namespace).Create(ctx, &event, metav1.CreateOptions{})
	if err != nil {
		logger.Error(err)
	}
}
