package domain

import v1 "k8s.io/api/core/v1"

// PolicyTargets is used to match entities with the required fields specified by the policy
type PolicyTargets struct {
	Kind      []string            `json:"kind"`
	Label     []map[string]string `json:"label"`
	Namespace []string            `json:"namespace"`
}

// PolicyParameters defines a needed input in a policy
type PolicyParameters struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Default  interface{} `json:"default"`
	Required bool        `json:"required"`
}

// Policy represents a policy
type Policy struct {
	Name        string             `json:"name"`
	ID          string             `json:"id"`
	Code        string             `json:"code"`
	Enable      string             `json:"enable"`
	Parameters  []PolicyParameters `json:"parameters"`
	Targets     PolicyTargets      `json:"targets"`
	Description string             `json:"description"`
	HowToSolve  string             `json:"how_to_solve"`
	Category    string             `json:"category"`
	Tags        []string           `json:"tags"`
	Severity    string             `json:"severity"`
	Controls    []string           `json:"controls"`
	Reference   interface{}        `json:"-"`
	GitCommit   string             `json:"git_commit,omitempty"`
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
		res[param.Name] = param.Default
	}
	return res
}
