package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	GroupName    = "magalix.com"
	GroupVersion = "v1"
	ResourceName = "policies"
)

var (
	SchemeGroupVersion         = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
	SchemeGroupVersionResource = schema.GroupVersionResource{Resource: ResourceName, Version: GroupVersion, Group: GroupName}
	SchemeBuilder              = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme                = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Policy{},
		&PolicyList{},
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
