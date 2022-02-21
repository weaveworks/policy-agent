package k8s

import (
	"context"

	"github.com/MagalixCorp/magalix-policy-agent/clients/kube"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixTechnologies/core/logger"
	corev1 "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var IgnoredPoliciesResource = schema.GroupVersionResource{Resource: "policies", Version: "v1", Group: "magalix.com"}

// GetEntitiesSources returns entities sources based on allowed list permissions
func GetEntitiesSources(ctx context.Context, kubeClient *kube.KubeClient) ([]domain.EntitiesSource, error) {
	var sources []domain.EntitiesSource
	permissions, err := kubeClient.GetAgentPermissions(ctx)
	if err != nil {
		return nil, err
	}
	rules := permissions.Status.ResourceRules
	allowedResources := make(map[string]struct{})
	for i := range rules {
		rule := rules[i]
		allowList := false
		for k := range rule.Verbs {
			if rule.Verbs[k] == "list" || rule.Verbs[k] == "*" {
				allowList = true
				break
			}
		}
		if allowList {
			for k := range rule.Resources {
				allowedResources[rule.Resources[k]] = struct{}{}
			}
		}
	}

	apiResourceList, err := kubeClient.GetAPIResources(ctx)
	if err != nil {
		return nil, err
	}

	_, checkAll := allowedResources["*"]
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
				if apiResource.Verbs[j] == "list" {
					foundList = true
					break
				}
			}
			if !foundList {
				continue
			}
			_, checkexplicit := allowedResources[apiResource.Name]
			if checkexplicit || checkAll {
				resource := schema.GroupVersionResource{
					Group:    groupVersion.Group,
					Version:  groupVersion.Version,
					Resource: apiResource.Name}
				if resource.String() == IgnoredPoliciesResource.String() {
					continue
				}

				sources = append(sources, &K8SEntitySource{
					resource:   resource,
					kubeClient: kubeClient,
					kind:       apiResource.Kind,
				})
			}
		}

	}
	return sources, nil
}

// K8SEntitySource retrieves specific kind of kubernetes resources
type K8SEntitySource struct {
	resource   schema.GroupVersionResource
	kubeClient *kube.KubeClient
	kind       string
}

// List returns list of resources from the entities srouce
func (k *K8SEntitySource) List(ctx context.Context, listOptions *domain.ListOptions) (*domain.EntitiesList, error) {
	metaListOptions := meta.ListOptions{
		Limit:    int64(listOptions.Limit),
		Continue: listOptions.KeySet,
	}
	entitiesList, err := k.kubeClient.List(ctx, k.resource, corev1.NamespaceAll, metaListOptions)
	if err != nil {
		return nil, err
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
