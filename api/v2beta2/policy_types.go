/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2beta2

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PolicyResourceName       = "policies"
	PolicyKind               = "Policy"
	PolicySetResourceName    = "policysets"
	PolicySetKind            = "PolicySet"
	PolicyConfigResourceName = "policyconfigs"
	PolicyConfigKind         = "PolicyConfig"
)

var (
	PolicyGroupVersionResource       = GroupVersion.WithResource(PolicyResourceName)
	PolicySetGroupVersionResource    = GroupVersion.WithResource(PolicySetResourceName)
	PolicyConfigGroupVersionResource = GroupVersion.WithResource(PolicyConfigResourceName)
)

// PolicyParameters defines a needed input in a policy
type PolicyParameters struct {
	// Name is a descriptive name of a policy parameter
	Name string `json:"name"`
	// Type is the type of that parameter, integer, string,...
	Type string `json:"type"`
	// Required specifies if this is a necessary value or not
	Required bool `json:"required"`
	// +optional
	// Value is the value for that parameter
	Value *apiextensionsv1.JSON `json:"value,omitempty"`
}

// PolicyTargets are filters used to determine which resources should be evaluated against a policy
type PolicyTargets struct {
	// Kinds is a list of Kubernetes kinds that are supported by this policy
	Kinds []string `json:"kinds"`
	// +optional
	// Labels is a list of Kubernetes labels that are needed to evaluate the policy against a resource
	// this filter is statisfied if only one label existed, using * for value make it so it will match if the key exists regardless of its value
	Labels []map[string]string `json:"labels"`
	// +optional
	// Namespaces is a list of Kubernetes namespaces that a resource needs to be a part of to evaluate against this policy
	Namespaces []string `json:"namespaces"`
}

type PolicyStandard struct {
	// ID idenitifer of the standarad
	ID string `json:"id"`
	// Controls standard controls
	Controls []string `json:"controls,omitempty"`
}

// PolicySpec defines the desired state of Policy
// It describes all that is needed to evaluate a resource against a rego code
// +kubebuilder:object:generate:true
type PolicySpec struct {
	// Name is the policy name
	Name string `json:"name"`
	// ID is the policy unique identifier
	ID string `json:"id"`
	// Code contains the policy rego code
	Code string `json:"code"`
	// +optional
	// Enabled flag for third parties consumers that indicates if this policy should be considered or not
	Enabled bool `json:"enabled,omitempty"`
	// +optional
	// Parameters are the inputs needed for the policy validation
	Parameters []PolicyParameters `json:"parameters,omitempty"`
	// +optional
	// Targets describes the required metadata that needs to be matched to evaluate a resource against the policy
	// all values specified need to exist in the resource to be considered for evaluation
	Targets PolicyTargets `json:"targets,omitempty"`
	// Description is a summary of what that policy validates
	Description string `json:"description"`
	// HowToSolve is a description of the steps required to solve the issues reported by the policy
	HowToSolve string `json:"how_to_solve"`
	// Category specifies under which grouping this policy should be included
	Category string `json:"category"`
	// +optional
	// Tags is a list of tags associated with that policy
	Tags []string `json:"tags,omitempty"`
	// +kubebuilder:validation:Enum=low;medium;high
	// Severity is a measure of the impact of that policy, can be low, medium or high
	Severity string `json:"severity"`
	// +optional
	// Standards is a list of policy standards that this policy falls under
	Standards []PolicyStandard `json:"standards"`
	//+optional
	//+kubebuilder:default:=kubernetes
	//+kubebuilder:validation:Enum=kubernetes;terraform
	// Provider policy provider, can be kubernetes, terraform
	Provider string `json:"provider"`
}

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

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Severity",type=string,JSONPath=`.spec.severity`
//+kubebuilder:printcolumn:name="Category",type=string,JSONPath=`.spec.category`
//+kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`

// Policy is the Schema for the policies API
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PolicySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// PolicyList contains a list of Policy
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Policy `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion

// PolicySet is the Schema for the policysets API
type PolicySet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PolicySetSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// PolicySetList contains a list of PolicySet
type PolicySetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicySet `json:"items"`
}

type PolicyConfigConfig struct {
	Parameters map[string]apiextensionsv1.JSON `json:"parameters"`
}

type PolicyConfigTarget struct {
	// +optional
	Kind string `json:"kind,omitempty"`
	// +optional
	Name string `json:"name,omitempty"`
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

func (t *PolicyConfigTarget) Type() string {
	if t.Name != "" && t.Kind != "" && t.Namespace != "" {
		return "resource"
	}
	if t.Namespace != "" {
		return "namespace"
	}
	return "cluster"
}

type PolicyConfigSpec struct {
	Config map[string]PolicyConfigConfig `json:"config"`
	Target PolicyConfigTarget            `json:"target"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion

// PolicyConfig is the Schema for the policyconfigs API
type PolicyConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PolicyConfigSpec `json:"spec,omitempty"`
}

func (pc *PolicyConfig) Validate() error {
	if pc.Spec.Target.Name != "" {
		if pc.Spec.Target.Kind == "" {
			return fmt.Errorf("kind is required when targeting specific resource")
		}
		if pc.Spec.Target.Namespace == "" {
			return fmt.Errorf("namespace is required when targeting specific resource")
		}
	}
	if pc.Spec.Target.Kind != "" {
		if pc.Spec.Target.Name == "" {
			return fmt.Errorf("name is required when targeting specific resource")
		}
		if pc.Spec.Target.Namespace == "" {
			return fmt.Errorf("namespace is required when targeting specific resource")
		}
	}

	if pc.Spec.Target.Name != "" && pc.Spec.Target.Kind != "" {
		if pc.Spec.Target.Labels != nil {
			return fmt.Errorf("cannot use labels when targeting specific resource")
		}
	}
	return nil
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// PolicyConfigList contains a list of PolicyConfig
type PolicyConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicyConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&Policy{},
		&PolicyList{},
		&PolicySet{},
		&PolicySetList{},
		&PolicyConfig{},
		&PolicyConfigList{},
	)
}
