package domain

type ConfigMatchApplication struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ConfigMatchResource struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type PolicyConfigMatch struct {
	Namespaces   []string                 `json:"namespaces,omitempty"`
	Applications []ConfigMatchApplication `json:"apps,omitempty"`
	Resources    []ConfigMatchResource    `json:"resources,omitempty"`
}

type PolicyConfigParameter struct {
	Value     interface{}
	ConfigRef string
}

type PolicyConfigConfig struct {
	Parameters map[string]PolicyConfigParameter `json:"parameters"`
}

// PolicyConfig represents a policy config
type PolicyConfig struct {
	Config map[string]PolicyConfigConfig `json:"config"`
	Match  PolicyConfigMatch             `json:"match"`
}
