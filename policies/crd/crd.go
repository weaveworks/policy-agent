package crd

import (
	"context"
	"fmt"
	"time"

	policiesCRDclient "github.com/MagalixCorp/magalix-policy-agent/clients/magalix.com/v1"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixTechnologies/core/logger"
	"k8s.io/client-go/tools/cache"
)

const syncPeriod = 5 * time.Minute

type PoliciesWatcher struct {
	informer *policiesCRDclient.PoliciesInformer
}

// NewPoliciesWatcher returns a policies source that fetches them from Kubernetes API
func NewPoliciesWatcher(client *policiesCRDclient.KubePoliciesClient) (*PoliciesWatcher, error) {
	informer := policiesCRDclient.NewPoliciesInformer(client, cache.ResourceEventHandlerFuncs{}, syncPeriod)
	err := informer.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start policies watcher informer, %w", err)
	}
	return &PoliciesWatcher{informer: informer}, nil
}

// Close stops the policies client informer
func (p *PoliciesWatcher) Close() {
	p.informer.Stop()
}

// GetAll returns all policies, implements github.com/MagalixCorp/magalix-policy-agent/pkg/domain.PoliciesSource
func (p *PoliciesWatcher) GetAll(_ context.Context) ([]domain.Policy, error) {
	var policies []domain.Policy
	policiesCRD := p.informer.List()
	logger.Debugw("retrieved CRD policies from cache", "count", len(policiesCRD))
	for i := range policiesCRD {
		policies = append(policies, policiesCRD[i].Spec)
	}
	return policies, nil
}
