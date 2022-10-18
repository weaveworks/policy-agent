package v2beta2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	PolicySetResourceName = "policysets"
	PolicySetKind         = "PolicySet"
)

var (
	PolicySetGroupVersionResource = GroupVersion.WithResource(PolicySetResourceName)
)

type PolicySetFilters struct {
	IDs        []string `json:"ids,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Severities []string `json:"severities,omitempty"`
	Standards  []string `json:"standards,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

type PolicySetSpec struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Filters PolicySetFilters `json:"filters"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion

// PolicySet is the Schema for the policysets API
type PolicySet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PolicySetSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion

// PolicySetList contains a list of PolicySet
type PolicySetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicySet `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&PolicySet{},
		&PolicySetList{},
	)
}
