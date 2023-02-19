package crd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MagalixTechnologies/policy-core/domain"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	"github.com/weaveworks/policy-agent/internal/utils"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	tenantLabel = "toolkit.fluxcd.io/tenant"
)

func (p *PoliciesWatcher) GetPolicyConfig(ctx context.Context, entity domain.Entity) (*domain.PolicyConfig, error) {
	configs := pacv2.PolicyConfigList{}
	err := p.cache.List(ctx, &configs)
	if err != nil {
		return nil, err
	}

	var workspaces, namespaces, apps, appsWithNamespace, resources, resourcesWithNamespace []pacv2.PolicyConfig

	var entityWorkspace string

	if entity.Namespace != "" {
		var ns v1.Namespace
		if err := p.cache.Get(ctx, client.ObjectKey{Name: entity.Namespace}, &ns); err != nil {
			return nil, fmt.Errorf("failed to get entity namespace: %w", err)
		}
		entityWorkspace = ns.GetLabels()[tenantLabel]
	}

	for _, config := range configs.Items {
		if entityWorkspace != "" {
			for _, workspace := range config.Spec.Match.Workspaces {
				if workspace == entityWorkspace {
					workspaces = append(workspaces, config)
					break
				}
			}
		}

		for _, namespace := range config.Spec.Match.Namespaces {
			if namespace == entity.Namespace {
				namespaces = append(namespaces, config)
				break
			}
		}

		if fluxApp := utils.GetFluxObject(entity.Labels); fluxApp != nil {
			for _, app := range config.Spec.Match.Applications {
				if app.Name == fluxApp.GetName() && app.Kind == fluxApp.GetKind() {
					if app.Namespace == "" {
						apps = append(apps, config)
						break
					} else if app.Namespace == fluxApp.GetNamespace() {
						appsWithNamespace = append(appsWithNamespace, config)
						break
					}
				}
			}
		}

		for _, resource := range config.Spec.Match.Resources {
			if resource.Name == entity.Name && resource.Kind == entity.Kind {
				if resource.Namespace != "" {
					if resource.Namespace == entity.Namespace {
						resourcesWithNamespace = append(resourcesWithNamespace, config)
					}
				} else {
					resources = append(resources, config)
				}
				break
			}
		}
	}

	allConfigs := []pacv2.PolicyConfig{}
	allConfigs = append(allConfigs, workspaces...)
	allConfigs = append(allConfigs, namespaces...)
	allConfigs = append(allConfigs, apps...)
	allConfigs = append(allConfigs, appsWithNamespace...)
	allConfigs = append(allConfigs, resources...)
	allConfigs = append(allConfigs, resourcesWithNamespace...)

	return override(allConfigs)
}

func override(configs []pacv2.PolicyConfig) (*domain.PolicyConfig, error) {
	configCRD := pacv2.PolicyConfig{
		Spec: pacv2.PolicyConfigSpec{
			Config: make(map[string]pacv2.PolicyConfigConfig),
		},
	}

	// store last policy config used to bind parameters
	confHistory := map[string]map[string]string{}
	for _, config := range configs {
		for policyID, policyConfig := range config.Spec.Config {
			// if no policy config exists, initialize a new one
			if _, ok := configCRD.Spec.Config[policyID]; !ok {
				configCRD.Spec.Config[policyID] = pacv2.PolicyConfigConfig{
					Parameters: make(map[string]apiextensionsv1.JSON),
				}
			}
			for k, v := range policyConfig.Parameters {
				// override policy parameter value
				configCRD.Spec.Config[policyID].Parameters[k] = v
				if _, ok := confHistory[policyID]; !ok {
					confHistory[policyID] = make(map[string]string)
				}
				// store policy config name to config hisotry
				confHistory[policyID][k] = config.GetName()
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
