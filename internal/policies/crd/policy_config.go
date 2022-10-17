package crd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MagalixTechnologies/policy-core/domain"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	"github.com/weaveworks/policy-agent/internal/utils"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
