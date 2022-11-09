package v2beta2

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	PolicySetResourceName    = "policysets"
	PolicySetKind            = "PolicySet"
	PolicySetListKind        = "PolicySetList"
	PolicySetAuditMode       = "audit"
	PolicySetAdmissionMode   = "admission"
	PolicySetTFAdmissionMode = "tf-admission"
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
	//+optional
	Name string `json:"name"`
	//+kubebuilder:validation:Enum=audit;admission;tf-admission
	// Mode is the policy set mode, must be one of audit,admission,tf-admission
	Mode    string           `json:"mode"`
	Filters PolicySetFilters `json:"filters"`
}

//+kubebuilder:object:root=true
//+kubebuilder:printcolumn:name="Mode",type=string,JSONPath=`.spec.mode`
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:storageversion

// PolicySet is the Schema for the policysets API
type PolicySet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PolicySetSpec `json:"spec,omitempty"`
}

// Match check if policy matches the policyset or not
func (ps *PolicySet) Match(policy Policy) bool {
	if len(ps.Spec.Filters.IDs) > 0 {
		for _, id := range ps.Spec.Filters.IDs {
			if policy.Name == id {
				return true
			}
		}
	}
	if len(ps.Spec.Filters.Categories) > 0 {
		for _, category := range ps.Spec.Filters.Categories {
			if policy.Spec.Category == category {
				return true
			}
		}
	}
	if len(ps.Spec.Filters.Severities) > 0 {
		for _, severity := range ps.Spec.Filters.Severities {
			if policy.Spec.Severity == severity {
				return true
			}
		}
	}
	if len(ps.Spec.Filters.Standards) > 0 {
		standards := map[string]struct{}{}
		for _, standard := range ps.Spec.Filters.Standards {
			standards[standard] = struct{}{}
		}
		for _, standard := range policy.Spec.Standards {
			if _, ok := standards[standard.ID]; ok {
				return true
			}
		}
	}
	if len(ps.Spec.Filters.Tags) > 0 {
		tags := map[string]struct{}{}
		for _, tag := range ps.Spec.Filters.Tags {
			tags[tag] = struct{}{}
		}
		for _, tag := range policy.Spec.Tags {
			if _, ok := tags[tag]; ok {
				return true
			}
		}
	}
	return false
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
