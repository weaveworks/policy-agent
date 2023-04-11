package domain

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

// Entity represents a kubernetes resource
type Entity struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	APIVersion      string                 `json:"apiVersion"`
	Kind            string                 `json:"kind"`
	Namespace       string                 `json:"namespace"`
	Manifest        map[string]interface{} `json:"manifest"`
	ResourceVersion string                 `json:"resource_version"`
	Labels          map[string]string      `json:"-"`
	GitCommit       string                 `json:"-"`
	HasParent       bool                   `json:"has_parent"`
}

// ObjectRef returns the kubernetes object reference of the entity
func (e *Entity) ObjectRef() *v1.ObjectReference {
	return &v1.ObjectReference{
		APIVersion:      e.APIVersion,
		Kind:            e.Kind,
		UID:             types.UID(e.ID),
		Name:            e.Name,
		Namespace:       e.Namespace,
		ResourceVersion: e.ResourceVersion,
	}
}

// NewEntityFromSpec takes map representing a Kubernetes entity and parses it into Entity struct
func NewEntityFromSpec(entitySpec map[string]interface{}) Entity {
	kubeEntity := unstructured.Unstructured{Object: entitySpec}
	metadata := entitySpec["metadata"].(map[string]interface{})
	delete(metadata, "managedFields")
	return Entity{
		ID:              string(kubeEntity.GetUID()),
		Name:            kubeEntity.GetName(),
		APIVersion:      kubeEntity.GetAPIVersion(),
		Kind:            kubeEntity.GetKind(),
		Namespace:       kubeEntity.GetNamespace(),
		Manifest:        entitySpec,
		ResourceVersion: kubeEntity.GetResourceVersion(),
		Labels:          kubeEntity.GetLabels(),
		HasParent:       len(kubeEntity.GetOwnerReferences()) != 0,
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
