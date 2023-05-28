package crd

import (
	"context"
	"encoding/json"
	"fmt"

	pacv2 "github.com/weaveworks/weave-policy-agent/api/v2beta2"
	"github.com/weaveworks/weave-policy-agent/pkg/logger"
	"github.com/weaveworks/weave-policy-agent/pkg/policy-core/domain"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlCache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AdmissionMode      = "admission"
	AuditMode          = "audit"
	TFAdmissionMode    = "tf-admission"
	KubernetesProvider = "kubernetes"
	TerraformProvider  = "terraform"
)

type PoliciesWatcher struct {
	cache    ctrlCache.Cache
	Mode     string
	Provider string
}

// NewPoliciesWatcher returns a policies source that fetches them from Kubernetes API
func NewPoliciesWatcher(ctx context.Context, mgr ctrl.Manager, mode, provider string) (*PoliciesWatcher, error) {
	return &PoliciesWatcher{
		cache:    mgr.GetCache(),
		Mode:     mode,
		Provider: provider,
	}, nil
}

// GetAll returns all policies, implements github.com/weaveworks/weave-policy-agent/pkg/policy-core/domain.PoliciesSource
func (p *PoliciesWatcher) GetAll(ctx context.Context) ([]domain.Policy, error) {
	policiesCRD := &pacv2.PolicyList{}
	err := p.cache.List(ctx, policiesCRD, &client.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error while retrieving policies CRD from cache: %w", err)
	}

	logger.Debugw("retrieved CRD policies from cache", "count", len(policiesCRD.Items))

	var policySets pacv2.PolicySetList
	err = p.cache.List(ctx, &policySets)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving policy sets CRD from cache: %w", err)
	}

	logger.Infow("retrieved CRD policy sets from cache", "count", len(policySets.Items))

	var policies []domain.Policy
	for i := range policiesCRD.Items {
		if !p.match(policiesCRD.Items[i], policySets) {
			continue
		}

		policyCRD := policiesCRD.Items[i].Spec
		policy := domain.Policy{
			Name:    policyCRD.Name,
			ID:      policyCRD.ID,
			Code:    policyCRD.Code,
			Enabled: policyCRD.Enabled,
			Targets: domain.PolicyTargets{
				Kinds:      policyCRD.Targets.Kinds,
				Labels:     policyCRD.Targets.Labels,
				Namespaces: policyCRD.Targets.Namespaces,
			},
			Description: policyCRD.Description,
			HowToSolve:  policyCRD.HowToSolve,
			Category:    policyCRD.Category,
			Tags:        policyCRD.Tags,
			Severity:    policyCRD.Severity,
			Reference: v1.ObjectReference{
				APIVersion:      policiesCRD.Items[i].APIVersion,
				Kind:            policiesCRD.Items[i].Kind,
				UID:             policiesCRD.Items[i].UID,
				Name:            policiesCRD.Items[i].Name,
				Namespace:       policiesCRD.Items[i].Namespace,
				ResourceVersion: policiesCRD.Items[i].ResourceVersion,
			},
			Mutate: policyCRD.Mutate,
		}

		for _, standardCRD := range policyCRD.Standards {
			standard := domain.PolicyStandard{
				ID:       standardCRD.ID,
				Controls: standardCRD.Controls,
			}
			policy.Standards = append(policy.Standards, standard)
		}

		for k := range policyCRD.Parameters {
			paramCRD := policyCRD.Parameters[k]
			param := domain.PolicyParameters{
				Name:     paramCRD.Name,
				Type:     paramCRD.Type,
				Required: paramCRD.Required,
			}
			if paramCRD.Value != nil {
				err = json.Unmarshal(paramCRD.Value.Raw, &param.Value)
				if err != nil {
					logger.Errorw("failed to load policy parameter value", "error", err)
				}
			}
			policy.Parameters = append(policy.Parameters, param)
		}

		policies = append(policies, policy)
	}
	return policies, nil
}

func (p *PoliciesWatcher) match(policy pacv2.Policy, policySets pacv2.PolicySetList) bool {
	// check provider
	if policy.Spec.Provider != p.Provider {
		return false
	}

	// check if tenant policy when mode is admission
	if p.Mode == AdmissionMode {
		for _, tag := range policy.Spec.Tags {
			if tag == pacv2.TenancyTag {
				return true
			}
		}
	}

	var count int
	for _, policySet := range policySets.Items {
		// check only policy sets with same mode
		if policySet.Spec.Mode == p.Mode {
			if policySet.Match(policy) {
				return true
			}
			count++
		}
	}

	// if there are no policysets configured return true
	return count == 0
}
