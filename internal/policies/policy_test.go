package crd

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	pacv2 "github.com/weaveworks/weave-policy-agent/api/v2beta2"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetPolicies(t *testing.T) {
	policies := []pacv2.Policy{
		{
			TypeMeta: v1.TypeMeta{
				APIVersion: pacv2.GroupVersion.Identifier(),
				Kind:       pacv2.PolicyKind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "policy-1",
			},
			Spec: pacv2.PolicySpec{
				ID:       "policy-1",
				Provider: "kubernetes",
				Category: "category-x",
				Severity: "severity-x",
				Standards: []pacv2.PolicyStandard{
					{ID: "standard-x"},
				},
				Tags: []string{"tag-x"},
				Parameters: []pacv2.PolicyParameters{
					{
						Name:     "x",
						Type:     "string",
						Value:    &apiextensionsv1.JSON{Raw: []byte(`"test"`)},
						Required: true,
					},
				},
			},
		},
		{
			TypeMeta: v1.TypeMeta{
				APIVersion: pacv2.GroupVersion.Identifier(),
				Kind:       pacv2.PolicyKind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "policy-2",
			},
			Spec: pacv2.PolicySpec{
				ID:       "policy-2",
				Provider: "kubernetes",
				Category: "category-y",
				Severity: "severity-y",
				Standards: []pacv2.PolicyStandard{
					{ID: "standard-y"},
				},
				Tags: []string{"tag-y"},
			},
		},
		{
			TypeMeta: v1.TypeMeta{
				APIVersion: pacv2.GroupVersion.Identifier(),
				Kind:       pacv2.PolicyKind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "policy-3",
			},
			Spec: pacv2.PolicySpec{
				ID:       "policy-3",
				Provider: "kubernetes",
				Category: "category-z",
				Severity: "severity-z",
				Standards: []pacv2.PolicyStandard{
					{ID: "standard-z"},
				},
				Tags: []string{"tag-z", "tenancy"},
			},
		},
		{
			TypeMeta: v1.TypeMeta{
				APIVersion: pacv2.GroupVersion.Identifier(),
				Kind:       pacv2.PolicyKind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "policy-4",
			},
			Spec: pacv2.PolicySpec{
				ID:       "policy-4",
				Provider: "terraform",
				Category: "category-x",
				Severity: "severity-x",
				Standards: []pacv2.PolicyStandard{
					{ID: "standard-x"},
				},
				Tags: []string{"tag-x"},
			},
		},
		{
			TypeMeta: v1.TypeMeta{
				APIVersion: pacv2.GroupVersion.Identifier(),
				Kind:       pacv2.PolicyKind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: "policy-5",
			},
			Spec: pacv2.PolicySpec{
				ID:       "policy-5",
				Provider: "terraform",
				Category: "category-y",
				Severity: "severity-y",
				Standards: []pacv2.PolicyStandard{
					{ID: "standard-y"},
				},
				Tags: []string{"tag-y"},
			},
		},
	}

	cases := []struct {
		description      string
		policies         []pacv2.Policy
		policySets       []pacv2.PolicySet
		provider         string
		mode             string
		expectedPolicies []string
	}{
		{
			policies:         policies,
			policySets:       []pacv2.PolicySet{},
			mode:             AuditMode,
			provider:         KubernetesProvider,
			expectedPolicies: []string{"policy-1", "policy-2", "policy-3"},
		},
		{
			policies:         policies,
			policySets:       []pacv2.PolicySet{},
			mode:             AdmissionMode,
			provider:         KubernetesProvider,
			expectedPolicies: []string{"policy-1", "policy-2", "policy-3"},
		},
		{
			policies:         policies,
			policySets:       []pacv2.PolicySet{},
			mode:             TFAdmissionMode,
			provider:         TerraformProvider,
			expectedPolicies: []string{"policy-4", "policy-5"},
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAuditMode,
						Filters: pacv2.PolicySetFilters{
							IDs: []string{"policy-1"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAuditMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-1"},
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAuditMode,
						Filters: pacv2.PolicySetFilters{
							Categories: []string{"category-x"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAuditMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-1"},
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAuditMode,
						Filters: pacv2.PolicySetFilters{
							Severities: []string{"severity-x"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAuditMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-1"},
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAuditMode,
						Filters: pacv2.PolicySetFilters{
							Tags: []string{"tag-x"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAuditMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-1"},
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAuditMode,
						Filters: pacv2.PolicySetFilters{
							IDs:        []string{"policy-1"},
							Categories: []string{"category-y"},
							Tags:       []string{"random"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAuditMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-1", "policy-2"},
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAuditMode,
						Filters: pacv2.PolicySetFilters{
							IDs: []string{"policy-1"},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-2"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAuditMode,
						Filters: pacv2.PolicySetFilters{
							IDs: []string{"policy-2"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAuditMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-1", "policy-2"},
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAdmissionMode,
						Filters: pacv2.PolicySetFilters{
							IDs: []string{"policy-1"},
						},
					},
				},
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-2"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAdmissionMode,
						Filters: pacv2.PolicySetFilters{
							IDs: []string{"policy-2"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAdmissionMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-1", "policy-2", "policy-3"},
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAuditMode,
						Filters: pacv2.PolicySetFilters{
							IDs: []string{"policy-4"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAuditMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: nil,
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAdmissionMode,
						Filters: pacv2.PolicySetFilters{
							IDs: []string{"policy-4"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAdmissionMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-3"},
		},

		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAuditMode,
						Filters: pacv2.PolicySetFilters{
							IDs: []string{"policy-1"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAdmissionMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-1", "policy-2", "policy-3"},
		},
		{
			policies: policies,
			policySets: []pacv2.PolicySet{
				{
					ObjectMeta: v1.ObjectMeta{Name: "policyset-1"},
					Spec: pacv2.PolicySetSpec{
						Mode: pacv2.PolicySetAdmissionMode,
						Filters: pacv2.PolicySetFilters{
							IDs: []string{"policy-1"},
						},
					},
				},
			},
			mode:             pacv2.PolicySetAdmissionMode,
			provider:         pacv2.PolicyKubernetesProvider,
			expectedPolicies: []string{"policy-1", "policy-3"},
		},
	}

	for i := range cases {
		schema := runtime.NewScheme()
		pacv2.AddToScheme(schema)
		corev1.AddToScheme(schema)

		var items []runtime.Object
		for idx := range cases[i].policies {
			item := cases[i].policies[idx]
			items = append(items, &item)
		}

		for idx := range cases[i].policySets {
			item := cases[i].policySets[idx]
			items = append(items, &item)
		}

		cache := NewFakeCache(schema, items...)

		watcher := PoliciesWatcher{
			cache:    cache,
			Mode:     cases[i].mode,
			Provider: cases[i].provider,
		}

		policies, err := watcher.GetAll(context.Background())
		if err != nil {
			t.Error(err)
		}

		var ids []string
		for _, policy := range policies {
			ids = append(ids, policy.ID)
		}

		assert.Equal(t, ids, cases[i].expectedPolicies, fmt.Sprintf("testcase: #%d", i))
	}
}
