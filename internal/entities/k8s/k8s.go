package k8s

import (
	"context"
	"errors"
	"fmt"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
	magalixv1 "github.com/weaveworks/policy-agent/api/v1"
	"github.com/weaveworks/policy-agent/internal/clients/kube"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	fieldSelectors "k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	allAllowed         = "*"
	listVerb           = "list"
	entityMetadataName = "metadata.name"
)

type rulesCache struct {
	apiGroups     map[string]struct{}
	resources     map[string]struct{}
	resourceNames []string
}

func newRulesCache(resourceNames []string) rulesCache {
	cache := rulesCache{
		apiGroups:     make(map[string]struct{}),
		resources:     make(map[string]struct{}),
		resourceNames: resourceNames,
	}
	for i := range resourceNames {
		if resourceNames[i] == allAllowed {
			cache.resourceNames = []string{}
			break
		}
	}
	return cache
}

func checkAllowed(subject string, rules map[string]struct{}) bool {
	_, checkAll := rules[allAllowed]
	_, checkExplicit := rules[subject]
	return checkExplicit || checkAll
}

func getValidateRules(ctx context.Context, kubeClient *kube.KubeClient) ([]rulesCache, error) {
	permissions, err := kubeClient.GetAgentPermissions(ctx)
	if err != nil {
		return nil, err
	}
	var rulesCaches []rulesCache
	rules := permissions.Status.ResourceRules
	foundPolicicesRule := false
	for i := range rules {
		rule := rules[i]
		cache := newRulesCache(rule.ResourceNames)
		allowList := false
		for k := range rule.Verbs {
			if rule.Verbs[k] == listVerb || rule.Verbs[k] == allAllowed {
				allowList = true
				break
			}
		}
		if allowList {
			for k := range rule.Resources {
				cache.resources[rule.Resources[k]] = struct{}{}
			}
			for k := range rule.APIGroups {
				cache.apiGroups[rule.APIGroups[k]] = struct{}{}
			}
			rulesCaches = append(rulesCaches, cache)
			checkPoliciesResource := checkAllowed(magalixv1.ResourceName, cache.resources)
			checkPoliciesGroup := checkAllowed(magalixv1.GroupVersion.Group, cache.apiGroups)
			if checkPoliciesResource && checkPoliciesGroup {
				foundPolicicesRule = true
			}
		}
	}
	if !foundPolicicesRule {
		return nil, errors.New("missing magalix policices resource permissions")
	}
	return rulesCaches, nil
}

// GetEntitiesSources returns entities sources based on allowed list permissions
func GetEntitiesSources(ctx context.Context, kubeClient *kube.KubeClient) ([]domain.EntitiesSource, error) {
	rulesCaches, err := getValidateRules(ctx, kubeClient)
	if err != nil {
		return nil, err
	}

	apiResourceList, err := kubeClient.GetAPIResources(ctx)
	if err != nil {
		return nil, err
	}

	var sources []domain.EntitiesSource
	for i := range apiResourceList {
		list := apiResourceList[i]
		groupVersion, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			logger.Errorw(
				"failed to parse group version",
				"group-version", list.GroupVersion,
				"error", err,
			)
			continue
		}
		for k := range list.APIResources {
			apiResource := list.APIResources[k]
			foundList := false
			for j := range apiResource.Verbs {
				if apiResource.Verbs[j] == listVerb {
					foundList = true
					break
				}
			}
			if !foundList {
				continue
			}

			for j := range rulesCaches {
				cache := rulesCaches[j]
				groupAllowed := checkAllowed(groupVersion.Group, cache.apiGroups)
				resourceAllowed := checkAllowed(apiResource.Name, cache.resources)
				if groupAllowed && resourceAllowed {
					resource := schema.GroupVersionResource{
						Group:    groupVersion.Group,
						Version:  groupVersion.Version,
						Resource: apiResource.Name}
					if resource.String() == magalixv1.GroupVersionResource.String() {
						continue
					}

					sources = append(sources, &K8SEntitySource{
						resource:      resource,
						kubeClient:    kubeClient,
						kind:          apiResource.Kind,
						resourceNames: cache.resourceNames,
					})
					break
				}

			}

		}

	}
	return sources, nil
}

// K8SEntitySource allows retrieving of items of a specific group version resource
type K8SEntitySource struct {
	resource      schema.GroupVersionResource
	kubeClient    *kube.KubeClient
	kind          string
	resourceNames []string
}

// List returns list of resources from the entities source
func (k *K8SEntitySource) List(ctx context.Context, listOptions *domain.ListOptions) (*domain.EntitiesList, error) {
	metaListOptions := meta.ListOptions{
		Limit:    int64(listOptions.Limit),
		Continue: listOptions.KeySet,
	}
	var entitiesList *unstructured.UnstructuredList
	var err error
	if len(k.resourceNames) != 0 {
		var items []unstructured.Unstructured
		for i := range k.resourceNames {
			selector := fieldSelectors.OneTermEqualSelector(entityMetadataName, k.resourceNames[i])
			opts := meta.ListOptions{FieldSelector: selector.String()}
			entitiesList, err = k.kubeClient.ListResourceItems(ctx, k.resource, corev1.NamespaceAll, opts)
			if err != nil {
				return nil, fmt.Errorf("error while getting resource with name %s: %w", k.resourceNames[i], err)
			}
			items = append(items, entitiesList.Items...)
		}
		entitiesList.Items = items
	} else {
		entitiesList, err = k.kubeClient.ListResourceItems(ctx, k.resource, corev1.NamespaceAll, metaListOptions)
		if err != nil {
			return nil, err
		}
	}

	keySet := entitiesList.GetContinue()
	var data []domain.Entity

	for i := range entitiesList.Items {
		entity := domain.NewEntityFromSpec(entitiesList.Items[i].Object)
		data = append(data, entity)
	}
	return &domain.EntitiesList{
		HasNext: keySet != "",
		KeySet:  keySet,
		Data:    data,
	}, nil
}

// Kind indicates the k8s kind of the source
func (k *K8SEntitySource) Kind() string {
	return k.kind
}
