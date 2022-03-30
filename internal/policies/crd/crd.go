package crd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
	magalixcomv1 "github.com/weaveworks/policy-agent/api/v1"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlCache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PoliciesWatcher struct {
	cache ctrlCache.Cache
}

// NewPoliciesWatcher returns a policies source that fetches them from Kubernetes API
func NewPoliciesWatcher(ctx context.Context, mgr ctrl.Manager) (*PoliciesWatcher, error) {
	return &PoliciesWatcher{cache: mgr.GetCache()}, nil
}

// GetAll returns all policies, implements github.com/MagalixTechnologies/policy-core/domain.PoliciesSource
func (p *PoliciesWatcher) GetAll(ctx context.Context) ([]domain.Policy, error) {
	var policies []domain.Policy
	policiesCRD := &magalixcomv1.PolicyList{}
	err := p.cache.List(ctx, policiesCRD, &client.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error while retrieving policies CRD from cache: %w", err)
	}
	logger.Debugw("retrieved CRD policies from cache", "count", len(policiesCRD.Items))
	for i := range policiesCRD.Items {
		policyCRD := policiesCRD.Items[i].Spec
		policy := domain.Policy{
			Name:   policyCRD.Name,
			ID:     policyCRD.ID,
			Code:   policyCRD.Code,
			Enable: policyCRD.Enable,
			Targets: domain.PolicyTargets{
				Kind:      policyCRD.Targets.Kind,
				Label:     policyCRD.Targets.Label,
				Namespace: policyCRD.Targets.Namespace,
			},
			Description: policyCRD.Description,
			HowToSolve:  policyCRD.HowToSolve,
			Category:    policyCRD.Category,
			Tags:        policyCRD.Tags,
			Severity:    policyCRD.Severity,
			Controls:    policyCRD.Controls,
			Reference: v1.ObjectReference{
				APIVersion:      policiesCRD.Items[i].APIVersion,
				Kind:            policiesCRD.Items[i].Kind,
				UID:             policiesCRD.Items[i].UID,
				Name:            policiesCRD.Items[i].Name,
				Namespace:       policiesCRD.Items[i].Namespace,
				ResourceVersion: policiesCRD.Items[i].ResourceVersion,
			},
		}
		for k := range policyCRD.Parameters {
			paramCRD := policyCRD.Parameters[k]
			param := domain.PolicyParameters{
				Name:     paramCRD.Name,
				Type:     paramCRD.Type,
				Required: paramCRD.Required,
			}
			if paramCRD.Default != nil {
				err = json.Unmarshal(paramCRD.Default.Raw, &param.Default)
				if err != nil {
					logger.Errorw("failed to load policy parameter default value", "error", err)
				}
			}
			policy.Parameters = append(policy.Parameters, param)
		}
		policies = append(policies, policy)
	}
	return policies, nil
}
