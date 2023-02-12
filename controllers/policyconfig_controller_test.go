package controllers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/MagalixTechnologies/uuid-go"
	"github.com/stretchr/testify/assert"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestPolicyConfigController(t *testing.T) {
	client := fake.NewFakeClient()
	err := pacv2.AddToScheme(client.Scheme())
	if err != nil {
		t.Error(err)
	}

	scheme := runtime.NewScheme()
	err = pacv2.AddToScheme(scheme)
	if err != nil {
		t.Error(err)
	}

	decoder, err := admission.NewDecoder(scheme)
	if err != nil {
		t.Error(err)
	}

	controller := PolicyConfigController{
		Client:  client,
		decoder: decoder,
	}

	existingConfigs := []pacv2.PolicyConfig{
		{
			TypeMeta: v1.TypeMeta{
				APIVersion: pacv2.GroupVersion.Identifier(),
				Kind:       pacv2.PolicyConfigKind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: uuid.NewV4().String(),
			},
			Spec: pacv2.PolicyConfigSpec{
				Match: pacv2.PolicyConfigTarget{
					Namespaces: []string{"dev"},
				},
			},
		},
		{
			TypeMeta: v1.TypeMeta{
				APIVersion: pacv2.GroupVersion.Identifier(),
				Kind:       pacv2.PolicyConfigKind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: uuid.NewV4().String(),
			},
			Spec: pacv2.PolicyConfigSpec{
				Match: pacv2.PolicyConfigTarget{
					Namespaces: []string{"prod"},
				},
			},
		},
		{
			TypeMeta: v1.TypeMeta{
				APIVersion: pacv2.GroupVersion.Identifier(),
				Kind:       pacv2.PolicyConfigKind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: uuid.NewV4().String(),
			},
			Spec: pacv2.PolicyConfigSpec{
				Match: pacv2.PolicyConfigTarget{
					Applications: []pacv2.PolicyTargetApplication{
						{
							Kind: "HelmRelease",
							Name: "my-app-1",
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
				Name: uuid.NewV4().String(),
			},
			Spec: pacv2.PolicyConfigSpec{
				Match: pacv2.PolicyConfigTarget{
					Applications: []pacv2.PolicyTargetApplication{
						{
							Kind:      "HelmRelease",
							Name:      "my-app-2",
							Namespace: "default",
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
				Name: uuid.NewV4().String(),
			},
			Spec: pacv2.PolicyConfigSpec{
				Match: pacv2.PolicyConfigTarget{
					Resources: []pacv2.PolicyTargetResource{
						{
							Kind: "Deployment",
							Name: "my-deployment-1",
						},
					},
				},
			},
		},
	}

	cases := []struct {
		name   string
		config pacv2.PolicyConfig
		allow  bool
	}{
		{
			name: "target namespace not targeted before",
			config: pacv2.PolicyConfig{
				TypeMeta: v1.TypeMeta{
					APIVersion: pacv2.GroupVersion.Identifier(),
					Kind:       pacv2.PolicyConfigKind,
				},
				ObjectMeta: v1.ObjectMeta{
					Name: uuid.NewV4().String(),
				},
				Spec: pacv2.PolicyConfigSpec{
					Match: pacv2.PolicyConfigTarget{
						Namespaces: []string{"test"},
					},
				},
			},
			allow: true,
		},
		{
			name: "target namespace already targeted before",
			config: pacv2.PolicyConfig{
				TypeMeta: v1.TypeMeta{
					APIVersion: pacv2.GroupVersion.Identifier(),
					Kind:       pacv2.PolicyConfigKind,
				},
				ObjectMeta: v1.ObjectMeta{
					Name: uuid.NewV4().String(),
				},
				Spec: pacv2.PolicyConfigSpec{
					Match: pacv2.PolicyConfigTarget{
						Namespaces: []string{"dev"},
					},
				},
			},
			allow: false,
		},
		{
			name: "target multiple namespaces one of them already targeted before",
			config: pacv2.PolicyConfig{
				TypeMeta: v1.TypeMeta{
					APIVersion: pacv2.GroupVersion.Identifier(),
					Kind:       pacv2.PolicyConfigKind,
				},
				ObjectMeta: v1.ObjectMeta{
					Name: uuid.NewV4().String(),
				},
				Spec: pacv2.PolicyConfigSpec{
					Match: pacv2.PolicyConfigTarget{
						Namespaces: []string{"dev", "test"},
					},
				},
			},
			allow: false,
		},
		{
			name: "target app without namespace already targeted before",
			config: pacv2.PolicyConfig{
				TypeMeta: v1.TypeMeta{
					APIVersion: pacv2.GroupVersion.Identifier(),
					Kind:       pacv2.PolicyConfigKind,
				},
				ObjectMeta: v1.ObjectMeta{
					Name: uuid.NewV4().String(),
				},
				Spec: pacv2.PolicyConfigSpec{
					Match: pacv2.PolicyConfigTarget{
						Applications: []pacv2.PolicyTargetApplication{
							{
								Kind: "HelmRelease",
								Name: "my-app-1",
							},
						},
					},
				},
			},
			allow: false,
		},
		{
			name: "target app with namespace not targeted before",
			config: pacv2.PolicyConfig{
				TypeMeta: v1.TypeMeta{
					APIVersion: pacv2.GroupVersion.Identifier(),
					Kind:       pacv2.PolicyConfigKind,
				},
				ObjectMeta: v1.ObjectMeta{
					Name: uuid.NewV4().String(),
				},
				Spec: pacv2.PolicyConfigSpec{
					Match: pacv2.PolicyConfigTarget{
						Applications: []pacv2.PolicyTargetApplication{
							{
								Kind:      "HelmRelease",
								Name:      "my-app-1",
								Namespace: "flux-system",
							},
						},
					},
				},
			},
			allow: true,
		},
		{
			name: "target app with namespace already targeted before",
			config: pacv2.PolicyConfig{
				TypeMeta: v1.TypeMeta{
					APIVersion: pacv2.GroupVersion.Identifier(),
					Kind:       pacv2.PolicyConfigKind,
				},
				ObjectMeta: v1.ObjectMeta{
					Name: uuid.NewV4().String(),
				},
				Spec: pacv2.PolicyConfigSpec{
					Match: pacv2.PolicyConfigTarget{
						Applications: []pacv2.PolicyTargetApplication{
							{
								Kind:      "HelmRelease",
								Name:      "my-app-2",
								Namespace: "default",
							},
						},
					},
				},
			},
			allow: false,
		},
		{
			name: "target app without namespace not targeted before",
			config: pacv2.PolicyConfig{
				TypeMeta: v1.TypeMeta{
					APIVersion: pacv2.GroupVersion.Identifier(),
					Kind:       pacv2.PolicyConfigKind,
				},
				ObjectMeta: v1.ObjectMeta{
					Name: uuid.NewV4().String(),
				},
				Spec: pacv2.PolicyConfigSpec{
					Match: pacv2.PolicyConfigTarget{
						Applications: []pacv2.PolicyTargetApplication{
							{
								Kind: "HelmRelease",
								Name: "my-app-2",
							},
						},
					},
				},
			},
			allow: true,
		},
		{
			name: "target resource without namespace already targeted before",
			config: pacv2.PolicyConfig{
				TypeMeta: v1.TypeMeta{
					APIVersion: pacv2.GroupVersion.Identifier(),
					Kind:       pacv2.PolicyConfigKind,
				},
				ObjectMeta: v1.ObjectMeta{
					Name: uuid.NewV4().String(),
				},
				Spec: pacv2.PolicyConfigSpec{
					Match: pacv2.PolicyConfigTarget{
						Resources: []pacv2.PolicyTargetResource{
							{
								Kind: "Deployment",
								Name: "my-deployment-1",
							},
						},
					},
				},
			},
			allow: false,
		},
		{
			name: "target resource with namespace not targeted before",
			config: pacv2.PolicyConfig{
				TypeMeta: v1.TypeMeta{
					APIVersion: pacv2.GroupVersion.Identifier(),
					Kind:       pacv2.PolicyConfigKind,
				},
				ObjectMeta: v1.ObjectMeta{
					Name: uuid.NewV4().String(),
				},
				Spec: pacv2.PolicyConfigSpec{
					Match: pacv2.PolicyConfigTarget{
						Resources: []pacv2.PolicyTargetResource{
							{
								Kind:      "Deployment",
								Name:      "my-deployment-1",
								Namespace: "default",
							},
						},
					},
				},
			},
			allow: true,
		},
	}

	ctx := context.Background()

	for i := range existingConfigs {
		err := client.Create(ctx, &existingConfigs[i])
		if err != nil {
			t.Error(err)
		}
	}

	for i := range cases {
		config := cases[i].config
		js, _ := json.Marshal(config)
		response := controller.Handle(ctx, admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				Name: config.Name,
				Kind: v1.GroupVersionKind{
					Group:   pacv2.GroupVersion.Group,
					Version: config.GroupVersionKind().Version,
					Kind:    pacv2.PolicyConfigKind,
				},
				Resource: v1.GroupVersionResource{
					Group:    pacv2.GroupVersion.Group,
					Version:  config.GroupVersionKind().Version,
					Resource: pacv2.PolicyConfigResourceName,
				},
				RequestKind: &v1.GroupVersionKind{
					Group:   pacv2.GroupVersion.Group,
					Version: config.GroupVersionKind().Version,
					Kind:    pacv2.PolicyConfigKind,
				},
				RequestResource: &v1.GroupVersionResource{
					Group:    pacv2.GroupVersion.Group,
					Version:  config.GroupVersionKind().Version,
					Resource: pacv2.PolicyConfigResourceName,
				},
				Object:    runtime.RawExtension{Raw: js},
				Operation: admissionv1.Create,
			},
		})
		assert.Equal(t, response.Allowed, cases[i].allow, cases[i].name)
	}

}
