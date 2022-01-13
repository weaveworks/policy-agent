package crd

import (
	"context"
	"time"

	policiesCRDclient "github.com/MagalixCorp/new-magalix-agent/clients/magalix.com/v1"
	"github.com/MagalixCorp/new-magalix-agent/validation"
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

func (p *PoliciesCRD) GetPolicies(ctx context.Context) ([]validation.Policy, error) {
	var policies []validation.Policy
	policiesCRD := p.informer.List()
	for i := range policiesCRD {
		policies = append(policies, policiesCRD[i].GetSpec())
	}
	return policies, nil
}
