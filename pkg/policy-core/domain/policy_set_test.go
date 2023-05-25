package domain

import "testing"

func TestPolicySet(t *testing.T) {
	tests := []struct {
		Name      string
		Policy    Policy
		PolicySet PolicySet
		Match     bool
	}{
		{
			Name:   "ids",
			Policy: Policy{ID: "my-policy"},
			PolicySet: PolicySet{
				Filters: PolicySetFilters{
					IDs: []string{"my-policy"},
				},
			},
			Match: true,
		},
		{
			Name:   "categories",
			Policy: Policy{Category: "my-category"},
			PolicySet: PolicySet{
				Filters: PolicySetFilters{
					Categories: []string{"my-category"},
				},
			},
			Match: true,
		},
		{
			Name:   "severities",
			Policy: Policy{Severity: "high"},
			PolicySet: PolicySet{
				Filters: PolicySetFilters{
					Severities: []string{"high"},
				},
			},
			Match: true,
		},
		{
			Name:   "standards",
			Policy: Policy{Standards: []PolicyStandard{{ID: "pci-dss"}}},
			PolicySet: PolicySet{
				Filters: PolicySetFilters{
					Standards: []string{"pci-dss"},
				},
			},
			Match: true,
		},
		{
			Name:   "tags",
			Policy: Policy{Tags: []string{"my-tag"}},
			PolicySet: PolicySet{
				Filters: PolicySetFilters{
					Tags: []string{"my-tag"},
				},
			},
			Match: true,
		},
		{
			Name:   "not-match",
			Policy: Policy{Tags: []string{}},
			PolicySet: PolicySet{
				Filters: PolicySetFilters{
					IDs:        []string{"my-id"},
					Categories: []string{"my-category"},
					Severities: []string{"high"},
					Standards:  []string{"pci-dss"},
					Tags:       []string{"my-tag"},
				},
			},
			Match: false,
		},
	}

	for _, test := range tests {
		if match := test.PolicySet.Match(test.Policy); match != test.Match {
			t.Errorf("unexpected result of test: %s, expected: %v but found: %v", test.Name, test.Match, match)
		}
	}
}
