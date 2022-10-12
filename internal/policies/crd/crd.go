package crd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	"github.com/weaveworks/policy-agent/controllers"
	v1 "k8s.io/api/core/v1"
	k8sLabels "k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlCache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	Provider  string
	PolicySet string
}

type PoliciesWatcher struct {
	cache  ctrlCache.Cache
	config Config
}

type PolicyConfigs struct {
	ClusterScoped   []domain.PolicyConfig
	NamespaceScoped []domain.PolicyConfig
	ResourceScoped  []domain.PolicyConfig
}

// func (pc *PolicyConfigs) Config() domain.PolicyConfig {
// 	config := domain.PolicyConfig{}
// 	for i := range pc.ClusterScoped {
// 		for policyID, policyConfig := range pc.ClusterScoped[i].Config {
// 			config.Config[policyID] = domain.PolicyConfigConfig{}
// 			for k, v := range policyConfig.Parameters {
// 				config.Config[policyID].Parameters[k] = v
// 			}
// 		}
// 	}

// 	for i := range pc.NamespaceScoped {
// 		for policyID, policyConfig := range pc.NamespaceScoped[i].Config {
// 			config.Config[policyID] = domain.PolicyConfigConfig{}
// 			for k, v := range policyConfig.Parameters {
// 				config.Config[policyID].Parameters[k] = v
// 			}
// 		}
// 	}

// 	for i := range pc.ResourceScoped {
// 		for policyID, policyConfig := range pc.ResourceScoped[i].Config {
// 			config.Config[policyID] = domain.PolicyConfigConfig{}
// 			for k, v := range policyConfig.Parameters {
// 				config.Config[policyID].Parameters[k] = v
// 			}
// 		}
// 	}

// 	return config
// }

// NewPoliciesWatcher returns a policies source that fetches them from Kubernetes API
func NewPoliciesWatcher(ctx context.Context, mgr ctrl.Manager, config Config) (*PoliciesWatcher, error) {
	return &PoliciesWatcher{
		cache:  mgr.GetCache(),
		config: config,
	}, nil
}

// GetAll returns all policies, implements github.com/MagalixTechnologies/policy-core/domain.PoliciesSource
func (p *PoliciesWatcher) GetAll(ctx context.Context) ([]domain.Policy, error) {
	policiesCRD := &pacv2.PolicyList{}

	err := p.cache.List(ctx, policiesCRD, &client.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error while retrieving policies CRD from cache: %w", err)
	}

	var policySet *domain.PolicySet
	if p.config.PolicySet != "" {
		policySet, err = p.GetPolicySet(ctx, p.config.PolicySet)
		if err != nil {
			return nil, err
		}
	}

	logger.Debugw("retrieved CRD policies from cache", "count", len(policiesCRD.Items))

	var policies []domain.Policy
	for i := range policiesCRD.Items {
		if policiesCRD.Items[i].Spec.Provider != p.config.Provider {
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
		}

		for _, standardCRD := range policyCRD.Standards {
			standard := domain.PolicyStandard{
				ID:       standardCRD.ID,
				Controls: standardCRD.Controls,
			}
			policy.Standards = append(policy.Standards, standard)
		}

		if policySet != nil {
			if match := policySet.Match(policy); !match {
				continue
			}
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

func (p *PoliciesWatcher) GetPolicySet(ctx context.Context, id string) (*domain.PolicySet, error) {
	policySet := pacv2.PolicySet{}
	err := p.cache.Get(ctx, client.ObjectKey{Name: id}, &policySet)
	if err != nil {
		return nil, err
	}
	return &domain.PolicySet{
		ID:      policySet.Spec.ID,
		Name:    policySet.Spec.Name,
		Filters: domain.PolicySetFilters(policySet.Spec.Filters),
	}, nil
}

func (p *PoliciesWatcher) GetPolicyConfig(ctx context.Context, target domain.PolicyConfigTarget) (*domain.PolicyConfig, error) {
	clusterConfigs := pacv2.PolicyConfigList{}
	labels := target.Labels

	labels[controllers.TargetScopeLabel] = "cluster"
	fmt.Println("========================", "cluster", labels)

	err := p.cache.List(ctx, &clusterConfigs, &client.ListOptions{
		LabelSelector: k8sLabels.SelectorFromValidatedSet(labels),
	})
	if err != nil {
		return nil, err
	}

	logger.Infow("cluster configs", "count", len(clusterConfigs.Items))

	namespaceConfigs := pacv2.PolicyConfigList{}
	labels = target.Labels
	labels[controllers.TargetNamespaceLabel] = target.Namespace
	labels[controllers.TargetScopeLabel] = "namespace"
	fmt.Println("========================", "namespace", labels)

	err = p.cache.List(ctx, &namespaceConfigs, &client.ListOptions{
		LabelSelector: k8sLabels.SelectorFromValidatedSet(labels),
	})
	if err != nil {
		return nil, err
	}

	logger.Infow("namespace configs", "count", len(namespaceConfigs.Items))

	resourceConfig := pacv2.PolicyConfigList{}
	labels = target.Labels
	labels[controllers.TargetKindLabel] = target.Kind
	labels[controllers.TargetNameLabel] = target.Name
	labels[controllers.TargetNamespaceLabel] = target.Namespace
	labels[controllers.TargetScopeLabel] = "resource"
	fmt.Println("========================", "resource", labels)

	err = p.cache.List(ctx, &resourceConfig, &client.ListOptions{
		LabelSelector: k8sLabels.SelectorFromValidatedSet(labels),
	})
	if err != nil {
		return nil, err
	}

	logger.Infow("resource configs", "count", len(resourceConfig.Items))

	return override(clusterConfigs, namespaceConfigs, resourceConfig)
}

func override(cluster, namespace, resource pacv2.PolicyConfigList) (*domain.PolicyConfig, error) {
	configCRD := pacv2.PolicyConfig{}
	for i := range cluster.Items {
		for policyID, policyConfig := range cluster.Items[i].Spec.Config {
			configCRD.Spec.Config[policyID] = pacv2.PolicyConfigConfig{}
			for k, v := range policyConfig.Parameters {
				configCRD.Spec.Config[policyID].Parameters[k] = v
			}
		}
	}

	for i := range namespace.Items {
		for policyID, policyConfig := range namespace.Items[i].Spec.Config {
			configCRD.Spec.Config[policyID] = pacv2.PolicyConfigConfig{}
			for k, v := range policyConfig.Parameters {
				configCRD.Spec.Config[policyID].Parameters[k] = v
			}
		}
	}

	for i := range resource.Items {
		for policyID, policyConfig := range resource.Items[i].Spec.Config {
			configCRD.Spec.Config[policyID] = pacv2.PolicyConfigConfig{}
			for k, v := range policyConfig.Parameters {
				configCRD.Spec.Config[policyID].Parameters[k] = v
			}
		}
	}

	config := domain.PolicyConfig{}
	for policyID, policyConfig := range configCRD.Spec.Config {
		config.Config[policyID] = domain.PolicyConfigConfig{}
		for k, v := range policyConfig.Parameters {
			var value interface{}
			err := json.Unmarshal(v.Raw, &value)
			if err != nil {
				return nil, err
			}
			logger.Infow("overriding parameter", "policy", policyID, "param", k, "oldValue", v, "newValue", value)
			config.Config[policyID].Parameters[k] = value
		}
	}

	return &config, nil
}
