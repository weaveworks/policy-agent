package crd

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetAllPolicies(t *testing.T) {
	watcher := PoliciesWatcher{cache: &policyCache{}, config: Config{Provider: "testing"}}

	policies, err := watcher.GetAll(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 3, len(policies), "mismatch number of policies")

	setWatcher := PoliciesWatcher{cache: &policyCache{}, config: Config{Provider: "policyset", PolicySet: "my-policyset"}}
	policieWithSet, err := setWatcher.GetAll(context.Background())
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 2, len(policieWithSet), "mismatch number of policies")
}

type policyCache struct {
	informertest.FakeInformers
}

func (c *policyCache) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {

	policies := pacv2.PolicyList{
		Items: []pacv2.Policy{
			{
				Spec: pacv2.PolicySpec{
					ID:       "id-1",
					Name:     "policy-1",
					Severity: "high",
					Provider: "testing",
					Parameters: []pacv2.PolicyParameters{
						{
							Name: "param1",
							Type: "type1",
						},
					},
					Standards: []pacv2.PolicyStandard{
						{
							ID: "standard-1",
							Controls: []string{
								"control1",
								"control2",
							},
						},
					},
					Code:    "code",
					Enabled: true,
				},
			},
			{
				Spec: pacv2.PolicySpec{
					ID:       "id-2",
					Name:     "policy-2",
					Severity: "high",
					Provider: "testing",
					Parameters: []pacv2.PolicyParameters{
						{
							Name: "param2",
							Type: "type2",
						},
					},
					Standards: []pacv2.PolicyStandard{
						{
							ID: "standard-2",
							Controls: []string{
								"control1",
								"control2",
							},
						},
					},
					Code:    "code",
					Enabled: true,
				},
			},
			{
				Spec: pacv2.PolicySpec{
					ID:       "id-3",
					Name:     "policy-3",
					Severity: "high",
					Provider: "testing",
					Parameters: []pacv2.PolicyParameters{
						{
							Name: "param1",
							Type: "type1",
						},
					},
					Standards: []pacv2.PolicyStandard{
						{
							ID: "standard-1",
							Controls: []string{
								"control1",
								"control2",
							},
						},
					},
					Code:    "code",
					Enabled: true,
				},
			},
			{
				Spec: pacv2.PolicySpec{
					ID:       "id-4",
					Name:     "policy-4",
					Severity: "low",
					Provider: "policyset",
					Parameters: []pacv2.PolicyParameters{
						{
							Name: "param1",
							Type: "type1",
						},
					},
					Standards: []pacv2.PolicyStandard{
						{
							ID: "standard-1",
							Controls: []string{
								"control1",
								"control2",
							},
						},
					},
					Code:    "code",
					Enabled: true,
				},
			},
			{
				Spec: pacv2.PolicySpec{
					ID:       "id-5",
					Name:     "policy-5",
					Severity: "low",
					Provider: "policyset",
					Parameters: []pacv2.PolicyParameters{
						{
							Name: "param1",
							Type: "type1",
						},
					},
					Standards: []pacv2.PolicyStandard{
						{
							ID: "standard-1",
							Controls: []string{
								"control1",
								"control2",
							},
						},
					},
					Code:    "code",
					Enabled: true,
				},
			},
		},
	}
	reflect.ValueOf(list).Elem().Set(reflect.ValueOf(policies))
	return nil
}

func (c *policyCache) Get(ctx context.Context, key types.NamespacedName, obj client.Object) error {

	policySet := pacv2.PolicySet{
		Spec: pacv2.PolicySetSpec{
			ID:   "my-policyset",
			Name: "my-policyset",
			Filters: pacv2.PolicySetFilters{
				Severities: []string{
					"low",
				},
			},
		},
	}

	reflect.ValueOf(obj).Elem().Set(reflect.ValueOf(policySet))
	return nil
}
