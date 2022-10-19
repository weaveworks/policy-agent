package v2beta2

import (
	"fmt"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PolicyConfigResourceName = "policyconfigs"
	PolicyConfigKind         = "PolicyConfig"
)

var (
	PolicyConfigGroupVersionResource = GroupVersion.WithResource(PolicyConfigResourceName)
)

type PolicyTargetApplication struct {
	//+kubebuilder:validation:Enum=HelmRelease;Kustomization
	Kind string `json:"kind"`
	Name string `json:"name"`
	//+optional
	Namespace string `json:"namespace"`
}

func (at *PolicyTargetApplication) ID() string {
	return fmt.Sprintf("%s/%s:%s", strings.ToLower(at.Kind), at.Name, at.Namespace)
}

type PolicyTargetResource struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	// +optional
	Namespace string `json:"namespace"`
}

func (rt *PolicyTargetResource) ID() string {
	return fmt.Sprintf("%s/%s:%s", strings.ToLower(rt.Kind), rt.Name, rt.Namespace)
}

type PolicyConfigTarget struct {
	//+optional
	Namespaces []string `json:"namespaces,omitempty"`
	//+optional
	Applications []PolicyTargetApplication `json:"apps,omitempty"`
	//+optional
	Resources []PolicyTargetResource `json:"resources,omitempty"`
}

type PolicyConfigConfig struct {
	Parameters map[string]apiextensionsv1.JSON `json:"parameters"`
}

type PolicyConfigSpec struct {
	Config map[string]PolicyConfigConfig `json:"config"`
	Match  PolicyConfigTarget            `json:"match"`
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

func (c *PolicyConfig) Validate() error {
	var target string

	if c.Spec.Match.Namespaces != nil {
		target = "namespaces"
	}

	if c.Spec.Match.Applications != nil {
		if target != "" {
			return fmt.Errorf("cannot target %s and apps in same policy config", target)
		}
		target = "apps"
	}

	if c.Spec.Match.Resources != nil {
		if target != "" {
			return fmt.Errorf("cannot target %s and resources in same policy config", target)
		}
		target = "resources"
	}

	return nil
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:storageversion

// PolicyConfigList contains a list of PolicyConfig
type PolicyConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicyConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(
		&PolicyConfig{},
		&PolicyConfigList{},
	)
}
