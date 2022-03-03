package domain

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Entity represents a kubernetes resource
type Entity struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Kind            string                 `json:"kind"`
	Namespace       string                 `json:"namespace"`
	Manifest        map[string]interface{} `json:"manifest"`
	ResourceVersion string                 `json:"resource_version"`
	Labels          map[string]string      `json:"labels"`
	GitCommit       string                 `json:"git_commit,omitempty,"`
}

// NewEntityFromSpec takes map representing a Kubernetes entity and parses it into Entity struct
func NewEntityFromSpec(entitySpec map[string]interface{}) Entity {
	kubeEntity := unstructured.Unstructured{Object: entitySpec}
	return Entity{
		ID:              string(kubeEntity.GetUID()),
		Name:            kubeEntity.GetName(),
		Kind:            kubeEntity.GetKind(),
		Namespace:       kubeEntity.GetNamespace(),
		Manifest:        entitySpec,
		ResourceVersion: kubeEntity.GetResourceVersion(),
		Labels:          kubeEntity.GetLabels(),
	}
}

// EntitiesList a grouping of Entity objects
type EntitiesList struct {
	HasNext bool
	// KeySet used to fetch next batch of entities
	KeySet string
	Data   []Entity
}

// EntitiesSource responsible for fetching entities of a spcific K8s kind
type EntitiesSource interface {
	// List returns entities
	List(ctx context.Context, listOptions *ListOptions) (*EntitiesList, error)
	// Kind returns kind of entities it retrieves
	Kind() string
}

// ListOptions configures the wanted return of a list operation
type ListOptions struct {
	Limit  int
	KeySet string
}
