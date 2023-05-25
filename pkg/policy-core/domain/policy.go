package domain

import (
	v1 "k8s.io/api/core/v1"
)

// PolicyTargets is used to match entities with the required fields specified by the policy
type PolicyTargets struct {
	Kinds      []string            `json:"kinds"`
	Labels     []map[string]string `json:"labels"`
	Namespaces []string            `json:"namespaces"`
}

// PolicyParameters defines a needed input in a policy
type PolicyParameters struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
	Required  bool        `json:"required"`
	ConfigRef string      `json:"config_ref,omitempty"`
}

type PolicyStandard struct {
	ID       string   `json:"id"`
	Controls []string `json:"controls"`
}

// Policy represents a policy
type Policy struct {
	Name        string             `json:"name"`
	ID          string             `json:"id"`
	Code        string             `json:"code"`
	Enabled     bool               `json:"enabled"`
	Parameters  []PolicyParameters `json:"parameters"`
	Targets     PolicyTargets      `json:"targets"`
	Description string             `json:"description"`
	HowToSolve  string             `json:"how_to_solve"`
	Category    string             `json:"category"`
	Tags        []string           `json:"tags"`
	Severity    string             `json:"severity"`
	Standards   []PolicyStandard   `json:"standards"`
	Reference   interface{}        `json:"-"`
	GitCommit   string             `json:"git_commit,omitempty"`
	Modes       []string           `json:"modes"`
	Mutate      bool               `json:"mutate"`
}

// ObjectRef returns the kubernetes object reference of the policy
func (p *Policy) ObjectRef() *v1.ObjectReference {
	if obj, ok := p.Reference.(v1.ObjectReference); ok {
		return &obj
	}
	return nil
}

// GetParametersMap returns policy parameters as a map
func (p *Policy) GetParametersMap() map[string]interface{} {
	res := make(map[string]interface{})
	for _, param := range p.Parameters {
		res[param.Name] = param.Value
	}
	return res
}
