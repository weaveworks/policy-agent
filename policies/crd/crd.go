package crd

import (
	"context"
	"time"

	policiesCRDclient "github.com/MagalixCorp/magalix-policy-agent/clients/magalix.com/v1"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixTechnologies/core/logger"
	"k8s.io/client-go/tools/cache"
)

const syncPeriod = 5 * time.Minute

type PoliciesCRD struct {
	informer *policiesCRDclient.PoliciesInformer
}

func NewPoliciesCRD(client *policiesCRDclient.KubePoliciesClient) (*PoliciesCRD, error) {
	informer := policiesCRDclient.NewPoliciesInformer(client, cache.ResourceEventHandlerFuncs{}, syncPeriod)
	err := informer.Start()
	if err != nil {
		return nil, err
	}
	return &PoliciesCRD{informer: informer}, nil
}

func (p *PoliciesCRD) Close() {
	p.informer.Stop()
}

func (p *PoliciesCRD) GetAll(ctx context.Context) ([]domain.Policy, error) {
	var policies []domain.Policy
	policiesCRD := p.informer.List()
	logger.Debugw("retrieved CRD policies from cache", "count", len(policiesCRD))
	for i := range policiesCRD {
		policies = append(policies, policiesCRD[i].Spec)
	}
	return policies, nil
}
