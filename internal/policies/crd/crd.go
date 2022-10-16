package crd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	"github.com/weaveworks/policy-agent/internal/utils"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

func (p *PoliciesWatcher) GetPolicyConfig(ctx context.Context, entity domain.Entity) (*domain.PolicyConfig, error) {
	kind := entity.Kind
	name := entity.Name
	namespace := entity.Namespace

	app := utils.GetFluxObject(entity.Labels)
	if app != nil {
		kind = app.GetKind()
		name = app.GetName()
		namespace = app.GetNamespace()
	}

	configs := pacv2.PolicyConfigList{}
	err := p.cache.List(ctx, &configs, &client.ListOptions{Namespace: namespace})
	if err != nil {
		return nil, err
	}

	namespaceConfigs := []pacv2.PolicyConfig{}
	appConfigs := []pacv2.PolicyConfig{}

	for _, config := range configs.Items {
		if config.Spec.Match == nil {
			namespaceConfigs = append(namespaceConfigs, config)
			continue
		}
		for _, target := range config.Spec.Match {
			if target.Kind == kind && target.Name == name {
				appConfigs = append(appConfigs, config)
				break
			}
		}
	}

	return override(namespaceConfigs, appConfigs)
}

func override(namespaceConfigs, appConfigs []pacv2.PolicyConfig) (*domain.PolicyConfig, error) {
	configCRD := pacv2.PolicyConfig{
		Spec: pacv2.PolicyConfigSpec{
			Config: make(map[string]pacv2.PolicyConfigConfig),
		},
	}

	confHistory := map[string]map[string]string{}

	for _, config := range namespaceConfigs {
		for policyID, policyConfig := range config.Spec.Config {
			configCRD.Spec.Config[policyID] = pacv2.PolicyConfigConfig{
				Parameters: make(map[string]apiextensionsv1.JSON),
			}
			for k, v := range policyConfig.Parameters {
				configCRD.Spec.Config[policyID].Parameters[k] = v
				if _, ok := confHistory[policyID]; !ok {
					confHistory[policyID] = make(map[string]string)
				}
				confHistory[policyID][k] = fmt.Sprintf("%s/%s", config.GetNamespace(), config.GetName())
			}
		}
	}

	for _, config := range appConfigs {
		for policyID, policyConfig := range config.Spec.Config {
			configCRD.Spec.Config[policyID] = pacv2.PolicyConfigConfig{
				Parameters: make(map[string]apiextensionsv1.JSON),
			}
			for k, v := range policyConfig.Parameters {
				configCRD.Spec.Config[policyID].Parameters[k] = v
				if _, ok := confHistory[policyID]; !ok {
					confHistory[policyID] = make(map[string]string)
				}
				confHistory[policyID][k] = fmt.Sprintf("%s/%s", config.GetNamespace(), config.GetName())
			}
		}
	}

	config := domain.PolicyConfig{
		Config: make(map[string]domain.PolicyConfigConfig),
	}
	for policyID, policyConfig := range configCRD.Spec.Config {
		config.Config[policyID] = domain.PolicyConfigConfig{
			Parameters: make(map[string]domain.PolicyConfigParameter),
		}
		for k, v := range policyConfig.Parameters {
			var value interface{}
			err := json.Unmarshal(v.Raw, &value)
			if err != nil {
				return nil, err
			}

			config.Config[policyID].Parameters[k] = domain.PolicyConfigParameter{
				Value:     value,
				ConfigRef: confHistory[policyID][k],
			}
		}
	}

	return &config, nil
}
