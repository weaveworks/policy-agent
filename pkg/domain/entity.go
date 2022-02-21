package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Entity struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Kind            string                 `json:"kind"`
	Namespace       string                 `json:"namespace"`
	Manifest        map[string]interface{} `json:"spec"`
	ResourceVersion string                 `json:"resource_version"`
	Labels          map[string]string      `json:"labels"`
	GitCommit       string                 `json:"git_commit,omitempty,"`
}

// NewEntityFromStringSpec takes string representing a Kubernetes entity and parses it into Entity struct
func NewEntityFromStringSpec(entityStringSpec string) (Entity, error) {
	var entitySpec map[string]interface{}
	err := json.Unmarshal([]byte(entityStringSpec), &entitySpec)
	if err != nil {
		return Entity{}, fmt.Errorf("invalid string format, %w", err)
	}
	return NewEntityFromSpec(entitySpec), nil
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

type EntitiesList struct {
	HasNext bool
	KeySet  string
	Data    []Entity
}

type EntitiesSource interface {
	// List returns entities
	List(ctx context.Context, listOptions *ListOptions) (*EntitiesList, error)
	// Kind returns kind of entities it retireves
	Kind() string
}
type ListOptions struct {
	Limit  int
	KeySet string
}
