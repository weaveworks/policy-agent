package domain

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Entity struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Kind            string                 `json:"kind"`
	Namespace       string                 `json:"namespace"`
	Spec            map[string]interface{} `json:"spec"`
	ResourceVersion string                 `json:"resource_version"`
	Labels          map[string]string      `json:"labels"`
}

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
