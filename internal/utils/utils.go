package utils

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var fluxControllerKindMap = map[string]string{
	"helm.toolkit.fluxcd.io":      "HelmRelease",
	"kustomize.toolkit.fluxcd.io": "Kustomization",
}

func GetFluxObject(labels map[string]string) *unstructured.Unstructured {
	for apiVersion, kind := range fluxControllerKindMap {
		name, ok := labels[fmt.Sprintf("%s/name", apiVersion)]
		if !ok {
			continue
		}

		namespace, ok := labels[fmt.Sprintf("%s/namespace", apiVersion)]
		if !ok {
			continue
		}

		obj := unstructured.Unstructured{}
		obj.SetAPIVersion(apiVersion)
		obj.SetKind(kind)
		obj.SetNamespace(namespace)
		obj.SetName(name)

		return &obj
	}
	return nil
}
