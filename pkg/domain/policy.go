package domain

type PolicyTargets struct {
	Kind      []string            `json:"kind"`
	Label     []map[string]string `json:"label"`
	Namespace []string            `json:"namespace"`
}

type PolicyParameters struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Required bool   `json:"required"`
}

type Policy struct {
	Name        string             `json:"name"`
	ID          string             `json:"id"`
	Code        string             `json:"code"`
	Parameters  []PolicyParameters `json:"parameters"`
	Targets     PolicyTargets      `json:"targets"`
	Description string             `json:"description"`
	HowToSolve  string             `json:"how_to_solve"`
	Category    string             `json:"category"`
	Tags        []string           `json:"tags"`
	Severity    string             `json:"severity"`
}

func (p *Policy) GetParametersMap() map[string]interface{} {
	res := make(map[string]interface{})
	for _, param := range p.Parameters {
		res[param.Name] = param.Default
	}
	return res
}
