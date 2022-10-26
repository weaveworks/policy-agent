package crd

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/MagalixTechnologies/policy-core/domain"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
)

func TestGetPolicyConfig(t *testing.T) {
	cases := []struct {
		entity  domain.Entity
		configs pacv2.PolicyConfigList
		result  domain.PolicyConfig
	}{
		{
			entity: domain.Entity{
				Name:      "deployment-1",
				Namespace: "default",
			},
			result: domain.PolicyConfig{
				Config: map[string]domain.PolicyConfigConfig{
					"policy-1": {
						Parameters: map[string]domain.PolicyConfigParameter{
							"param-1": {
								Value:     float64(1),
								ConfigRef: "config-1",
							},
							"param-2": {
								Value:     float64(3),
								ConfigRef: "config-1",
							},
						},
					},
				},
			},
		},
		{
			entity: domain.Entity{
				Name:      "deployment-1",
				Namespace: "default",
				Labels: map[string]string{
					"kustomize.toolkit.fluxcd.io/name":      "my-app",
					"kustomize.toolkit.fluxcd.io/namespace": "kube-system",
				},
			},
			result: domain.PolicyConfig{
				Config: map[string]domain.PolicyConfigConfig{
					"policy-1": {
						Parameters: map[string]domain.PolicyConfigParameter{
							"param-1": {
								Value:     float64(2),
								ConfigRef: "config-2",
							},
							"param-2": {
								Value:     float64(3),
								ConfigRef: "config-1",
							},
						},
					},
				},
			},
		},
		{
			entity: domain.Entity{
				Name:      "deployment-1",
				Namespace: "default",
				Labels: map[string]string{
					"kustomize.toolkit.fluxcd.io/name":      "my-app",
					"kustomize.toolkit.fluxcd.io/namespace": "flux-system",
				},
			},
			result: domain.PolicyConfig{
				Config: map[string]domain.PolicyConfigConfig{
					"policy-1": {
						Parameters: map[string]domain.PolicyConfigParameter{
							"param-1": {
								Value:     float64(3),
								ConfigRef: "config-3",
							},
							"param-2": {
								Value:     float64(3),
								ConfigRef: "config-1",
							},
						},
					},
				},
			},
		},
		{
			entity: domain.Entity{
				Kind:      "Deployment",
				Name:      "deployment-2",
				Namespace: "test",
				Labels: map[string]string{
					"kustomize.toolkit.fluxcd.io/name":      "my-app",
					"kustomize.toolkit.fluxcd.io/namespace": "flux-system",
				},
			},
			result: domain.PolicyConfig{
				Config: map[string]domain.PolicyConfigConfig{
					"policy-1": {
						Parameters: map[string]domain.PolicyConfigParameter{
							"param-1": {
								Value:     float64(4),
								ConfigRef: "config-4",
							},
						},
					},
				},
			},
		},
		{
			entity: domain.Entity{
				Kind:      "Deployment",
				Name:      "deployment-2",
				Namespace: "default",
				Labels: map[string]string{
					"kustomize.toolkit.fluxcd.io/name":      "my-app",
					"kustomize.toolkit.fluxcd.io/namespace": "flux-system",
				},
			},
			result: domain.PolicyConfig{
				Config: map[string]domain.PolicyConfigConfig{
					"policy-1": {
						Parameters: map[string]domain.PolicyConfigParameter{
							"param-1": {
								Value:     float64(5),
								ConfigRef: "config-5",
							},
							"param-2": {
								Value:     float64(3),
								ConfigRef: "config-1",
							},
						},
					},
				},
			},
		},
	}

	watcher := PoliciesWatcher{cache: &cacheMock{
		items: map[reflect.Type]client.ObjectList{
			reflect.TypeOf(pacv2.PolicyConfigList{}): &pacv2.PolicyConfigList{
				Items: []pacv2.PolicyConfig{
					{
						TypeMeta: v1.TypeMeta{
							APIVersion: pacv2.GroupVersion.Identifier(),
							Kind:       pacv2.PolicyConfigKind,
						},
						ObjectMeta: v1.ObjectMeta{
							Name: "config-1",
						},
						Spec: pacv2.PolicyConfigSpec{
							Match: pacv2.PolicyConfigTarget{
								Namespaces: []string{"default"},
							},
							Config: map[string]pacv2.PolicyConfigConfig{
								"policy-1": {
									Parameters: map[string]apiextensionsv1.JSON{
										"param-1": {Raw: []byte("1")},
										"param-2": {Raw: []byte("3")},
									},
								},
							},
						},
					},
					{
						TypeMeta: v1.TypeMeta{
							APIVersion: pacv2.GroupVersion.Identifier(),
							Kind:       pacv2.PolicyConfigKind,
						},
						ObjectMeta: v1.ObjectMeta{
							Name: "config-2",
						},
						Spec: pacv2.PolicyConfigSpec{
							Match: pacv2.PolicyConfigTarget{
								Applications: []pacv2.PolicyTargetApplication{
									{
										Kind: "Kustomization",
										Name: "my-app",
									},
								},
							},
							Config: map[string]pacv2.PolicyConfigConfig{
								"policy-1": {
									Parameters: map[string]apiextensionsv1.JSON{
										"param-1": {Raw: []byte("2")},
									},
								},
							},
						},
					},
					{
						TypeMeta: v1.TypeMeta{
							APIVersion: pacv2.GroupVersion.Identifier(),
							Kind:       pacv2.PolicyConfigKind,
						},
						ObjectMeta: v1.ObjectMeta{
							Name: "config-3",
						},
						Spec: pacv2.PolicyConfigSpec{
							Match: pacv2.PolicyConfigTarget{
								Applications: []pacv2.PolicyTargetApplication{
									{
										Kind:      "Kustomization",
										Name:      "my-app",
										Namespace: "flux-system",
									},
								},
							},
							Config: map[string]pacv2.PolicyConfigConfig{
								"policy-1": {
									Parameters: map[string]apiextensionsv1.JSON{
										"param-1": {Raw: []byte("3")},
									},
								},
							},
						},
					},
					{
						TypeMeta: v1.TypeMeta{
							APIVersion: pacv2.GroupVersion.Identifier(),
							Kind:       pacv2.PolicyConfigKind,
						},
						ObjectMeta: v1.ObjectMeta{
							Name: "config-4",
						},
						Spec: pacv2.PolicyConfigSpec{
							Match: pacv2.PolicyConfigTarget{
								Resources: []pacv2.PolicyTargetResource{
									{
										Kind: "Deployment",
										Name: "deployment-2",
									},
								},
							},
							Config: map[string]pacv2.PolicyConfigConfig{
								"policy-1": {
									Parameters: map[string]apiextensionsv1.JSON{
										"param-1": {Raw: []byte("4")},
									},
								},
							},
						},
					},
					{
						TypeMeta: v1.TypeMeta{
							APIVersion: pacv2.GroupVersion.Identifier(),
							Kind:       pacv2.PolicyConfigKind,
						},
						ObjectMeta: v1.ObjectMeta{
							Name: "config-5",
						},
						Spec: pacv2.PolicyConfigSpec{
							Match: pacv2.PolicyConfigTarget{
								Resources: []pacv2.PolicyTargetResource{
									{
										Kind:      "Deployment",
										Name:      "deployment-2",
										Namespace: "default",
									},
								},
							},
							Config: map[string]pacv2.PolicyConfigConfig{
								"policy-1": {
									Parameters: map[string]apiextensionsv1.JSON{
										"param-1": {Raw: []byte("5")},
									},
								},
							},
						},
					},
				},
			},
		},
	}}
	for i := range cases {
		result, err := watcher.GetPolicyConfig(context.Background(), cases[i].entity)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, cases[i].result.Config, result.Config)
	}

}
