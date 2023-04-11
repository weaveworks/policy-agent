package domain

// PolicySetFilters defines a policy filters
type PolicySetFilters struct {
	IDs        []string `json:"ids"`
	Categories []string `json:"categories"`
	Severities []string `json:"severities"`
	Standards  []string `json:"standards"`
	Tags       []string `json:"tags"`
}

// PolicySet represents a policy set
type PolicySet struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Mode    string           `json:"mode"`
	Filters PolicySetFilters `json:"filters"`
}

// Match checks if the provided policy matches the policy set or not
func (ps *PolicySet) Match(policy Policy) bool {
	if len(ps.Filters.IDs) > 0 {
		for _, id := range ps.Filters.IDs {
			if policy.ID == id {
				return true
			}
		}
		return false
	}

	if len(ps.Filters.Categories) > 0 {
		for _, category := range ps.Filters.Categories {
			if policy.Category == category {
				return true
			}
		}
	}

	if len(ps.Filters.Severities) > 0 {
		for _, severity := range ps.Filters.Severities {
			if policy.Severity == severity {
				return true
			}
		}
	}

	if len(ps.Filters.Standards) > 0 {
		standards := map[string]struct{}{}
		for _, standard := range ps.Filters.Standards {
			standards[standard] = struct{}{}
		}
		for _, standard := range policy.Standards {
			if _, ok := standards[standard.ID]; ok {
				return true
			}
		}
	}

	if len(ps.Filters.Tags) > 0 {
		tags := map[string]struct{}{}
		for _, tag := range ps.Filters.Tags {
			tags[tag] = struct{}{}
		}
		for _, tag := range policy.Tags {
			if _, ok := tags[tag]; ok {
				return true
			}
		}
	}

	return false
}
