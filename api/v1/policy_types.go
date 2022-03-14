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

package v1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceName = "policies"
	Kind         = "Policy"
)

var GroupVersionResource = GroupVersion.WithResource(ResourceName)

// PolicyParameters defines a needed input in a policy
type PolicyParameters struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	// +optional
	Default *apiextensionsv1.JSON `json:"default"`
}

type PolicyTargets struct {
	Kind []string `json:"kind"`
	// +optional
	Label []map[string]string `json:"label"`
	// +optional
	Namespace []string `json:"namespace"`
}

// Policy represents a policy
//+kubebuilder:object:generate:true
type PolicySpec struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Code string `json:"code"`
	// +optional
	Enable string `json:"enable"`
	// +optional
	Parameters []PolicyParameters `json:"parameters"`
	// +optional
	Targets     PolicyTargets `json:"targets"`
	Description string        `json:"description"`
	HowToSolve  string        `json:"how_to_solve"`
	Category    string        `json:"category"`
	// +optional
	Tags     []string `json:"tags"`
	Severity string   `json:"severity"`
	// +optional
	Controls []string `json:"controls"`
}

// PolicySpec defines the desired state of Policy

// PolicyStatus defines the observed state of Policy
type PolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// Policy is the Schema for the policies API
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PolicySpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// PolicyList contains a list of Policy
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Policy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Policy{}, &PolicyList{})
}
