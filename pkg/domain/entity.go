package domain

import (
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Entity struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Kind            string                 `json:"kind"`
	Namespace       string                 `json:"namespace"`
	Spec            map[string]interface{} `json:"spec"`
	ResourceVersion string                 `json:"resource_version"`
	Labels          map[string]string      `json:"labels"`
}

// NewEntityByStringSpec takes string representing a Kubernetes entity and parses it into Entity struct
func NewEntityByStringSpec(entityStringSpec string) (Entity, error) {
	var entitySpec map[string]interface{}
	err := json.Unmarshal([]byte(entityStringSpec), &entitySpec)
	if err != nil {
		return Entity{}, fmt.Errorf("invalid string format, %w", err)
	}
	return NewEntityBySpec(entitySpec), nil
}

// NewEntityBySpec takes map representing a Kubernetes entity and parses it into Entity struct
func NewEntityBySpec(entitySpec map[string]interface{}) Entity {
	kubeEntity := unstructured.Unstructured{Object: entitySpec}
	return Entity{
		ID:              string(kubeEntity.GetUID()),
		Name:            kubeEntity.GetName(),
		Kind:            kubeEntity.GetKind(),
		Namespace:       kubeEntity.GetNamespace(),
		Spec:            entitySpec,
		ResourceVersion: kubeEntity.GetResourceVersion(),
		Labels:          kubeEntity.GetLabels(),
	}
}
