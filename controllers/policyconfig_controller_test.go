package controllers

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	"github.com/weaveworks/policy-agent/pkg/uuid-go"
	admissionv1 "k8s.io/api/admission/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestPolicyConfigValidator(t *testing.T) {
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

func TestPolicyConfigControllerReconciler(t *testing.T) {
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
	controller.InjectDecoder(decoder)

	policyData := []struct {
		id        string
		category  string
		severity  string
		provider  string
		standards []string
		tags      []string
	}{
		{
			id:        "policy-1",
			category:  "category-x",
			severity:  "low",
			provider:  pacv2.PolicyKubernetesProvider,
			standards: []string{"standard-x"},
			tags:      []string{"tag-x"},
		},
		{
			id:        "policy-2",
			category:  "category-y",
			severity:  "high",
			provider:  pacv2.PolicyKubernetesProvider,
			standards: []string{"standard-y"},
			tags:      []string{"tag-y"},
		},
	}
	policyConfigNames := []string{"policy-config-x", "policy-config-y", "policy-config-z"}

	existingConfigs := []pacv2.PolicyConfig{
		{
			TypeMeta: v1.TypeMeta{
				APIVersion: pacv2.GroupVersion.Identifier(),
				Kind:       pacv2.PolicyConfigKind,
			},
			ObjectMeta: v1.ObjectMeta{
				Name: policyConfigNames[0],
			},
			Spec: pacv2.PolicyConfigSpec{
				Match: pacv2.PolicyConfigTarget{
					Resources: []pacv2.PolicyTargetResource{
						{
							Kind: "Deployment",
							Name: "my-deployment-x",
						},
					},
				},
				Config: map[string]pacv2.PolicyConfigConfig{
					"policy-1": {
						Parameters: map[string]apiextensionsv1.JSON{
							"param1": {Raw: []byte{}},
						},
					},
					"policy-2": {
						Parameters: map[string]apiextensionsv1.JSON{
							"param1": {Raw: []byte{}},
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
				Name: policyConfigNames[1],
			},
			Spec: pacv2.PolicyConfigSpec{
				Match: pacv2.PolicyConfigTarget{
					Resources: []pacv2.PolicyTargetResource{
						{
							Kind: "Deployment",
							Name: "my-deployment-y",
						},
					},
				},
				Config: map[string]pacv2.PolicyConfigConfig{
					"policy-1": {
						Parameters: map[string]apiextensionsv1.JSON{
							"param1": {Raw: []byte{}},
						},
					},
					"policy-3": {
						Parameters: map[string]apiextensionsv1.JSON{
							"param1": {Raw: []byte{}},
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
				Name: policyConfigNames[2],
			},
			Spec: pacv2.PolicyConfigSpec{
				Match: pacv2.PolicyConfigTarget{
					Resources: []pacv2.PolicyTargetResource{
						{
							Kind: "Deployment",
							Name: "my-deployment-z",
						},
					},
				},
				Config: map[string]pacv2.PolicyConfigConfig{
					"policy-3": {
						Parameters: map[string]apiextensionsv1.JSON{
							"param1": {Raw: []byte{}},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	for _, item := range policyData {
		policy := createPolicy(
			item.id,
			item.category,
			item.severity,
			item.provider,
			item.standards,
			item.tags,
		)
		if err = client.Create(ctx, &policy); err != nil {
			t.Error(err)
		}
	}

	for i := range existingConfigs {
		err := client.Create(ctx, &existingConfigs[i])
		if err != nil {
			t.Error(err)
		}
		request := controllerruntime.Request{NamespacedName: types.NamespacedName{Name: existingConfigs[i].Name}}
		_, err = controller.Reconcile(ctx, request)
		if err != nil {
			t.Error(err)
		}
	}
	expectedStatus := map[string]string{
		policyConfigNames[0]: "OK",
		policyConfigNames[1]: "Warning",
		policyConfigNames[2]: "Warning",
	}
	expectedMissingPolicies := map[string][]string{
		policyConfigNames[0]: {},
		policyConfigNames[1]: {"policy-3"},
		policyConfigNames[2]: {"policy-3"},
	}

	policyConfigs := pacv2.PolicyConfigList{}

	if err = client.List(context.Background(), &policyConfigs); err != nil {
		t.Fatal(err)
	}

	for _, policyConfig := range policyConfigs.Items {
		assert.EqualValues(t, expectedStatus[policyConfig.Name], policyConfig.Status.Status)
		assert.ElementsMatch(t, expectedMissingPolicies[policyConfig.Name], policyConfig.Status.MissingPolicies)
	}
	reconcilationRequests := controller.reconcile(&policyConfigs.Items[0])
	for _, request := range reconcilationRequests {
		assert.Contains(t, policyConfigNames, request.Name)
	}
}
