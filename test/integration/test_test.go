package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/stretchr/testify/assert"
	"github.com/weaveworks/policy-agent/api/v2beta2"

	appsv1 "k8s.io/api/apps/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	minimumReplicaCountPolicy        = "weave.policies.containers-minimum-replica-count"
	containersInPrivilegedModePolicy = "weave.policies.containers-running-in-privileged-mode"
	missingOwnerLabelPolicy          = "weave.policies.missing-owner-label"
	testMutationDeployment           = "test-mutation-deployment"
)

func TestIntegration(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	assert.Nil(t, err)

	kubeConfigPath := filepath.Join(homeDir, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		panic(err)
	}

	cl, err := client.New(config, client.Options{})
	if err != nil {
		panic(err)
	}

	v2beta2.AddToScheme(cl.Scheme())

	ctx := context.Background()

	t.Run("check policies modes are correct", func(t *testing.T) {
		modes := map[string][]string{
			minimumReplicaCountPolicy:        {"admission"},
			containersInPrivilegedModePolicy: {"admission", "audit"},
			missingOwnerLabelPolicy:          {"audit"},
		}

		policies, err := listPolicies(ctx, cl)
		assert.Nil(t, err)

		assert.Equal(t, len(policies.Items), 3)

		for _, policy := range policies.Items {
			assert.ElementsMatch(t, modes[policy.GetName()], policy.Status.Modes)
		}
	})

	t.Run("check audit results", func(t *testing.T) {
		opts := []client.ListOption{
			client.MatchingFields{
				"reason":                   "PolicyViolation",
				"involvedObject.namespace": "default",
			},
			client.MatchingLabels{domain.PolicyValidationTypeLabel: "Audit"},
		}

		events, err := listViolationEvents(ctx, cl, opts)
		assert.Nil(t, err)

		assert.Equal(t, len(events.Items), 4)

		expected := map[string]int{
			missingOwnerLabelPolicy:          2,
			containersInPrivilegedModePolicy: 2,
		}

		actual := map[string]int{}
		for i := range events.Items {
			actual[events.Items[i].Related.Name]++
		}
		assert.ObjectsAreEqual(expected, actual)
	})

	t.Run("check admission results", func(t *testing.T) {
		err := kubectl("apply", "-f", "data/resources/admission_test_resources.yaml")
		assert.NotNil(t, err)

		time.Sleep(2 * time.Second)

		opts := []client.ListOption{
			client.MatchingFields{
				"reason": "PolicyViolation",
			},
			client.MatchingLabels{domain.PolicyValidationTypeLabel: "Admission"},
		}

		events, err := listViolationEvents(ctx, cl, opts)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, 8, len(events.Items))

		expected := map[string]struct {
			value     float64
			configRef string
		}{
			"orphan-deployment/default": {value: 3, configRef: "namespace-config"},
			"helm-app/flux-system":      {value: 4, configRef: "helm-app-config"},
			"kustomize-app/flux-system": {value: 5, configRef: "kustomize-app-config"},
			"test-deployment/default":   {value: 6, configRef: "resource-config"},
		}

		for i := range events.Items {
			result, err := domain.NewPolicyValidationFRomK8sEvent(&events.Items[i])
			if err != nil {
				t.Fatal(err)
			}

			if result.Policy.ID != minimumReplicaCountPolicy {
				continue
			}

			key := fmt.Sprintf("%s/%s", result.Entity.Name, result.Entity.Namespace)

			var value interface{}
			var configRef string

			for _, parameter := range result.Policy.Parameters {
				if parameter.Name == "replica_count" {
					value = parameter.Value
					configRef = parameter.ConfigRef
				}
				break
			}

			assert.Equal(t, expected[key].value, value)
			assert.Equal(t, expected[key].configRef, configRef)
		}
	})

	t.Run("create conflicting policy configs", func(t *testing.T) {
		configs := []v2beta2.PolicyConfig{
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v2beta2.GroupVersion.Identifier(),
					Kind:       v2beta2.PolicyConfigKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "namespace-config-conflict",
				},
				Spec: v2beta2.PolicyConfigSpec{
					Match: v2beta2.PolicyConfigTarget{
						Namespaces: []string{"default"},
					},
					Config: map[string]v2beta2.PolicyConfigConfig{
						minimumReplicaCountPolicy: {
							Parameters: map[string]apiextensionsv1.JSON{},
						},
					},
				},
			},
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v2beta2.GroupVersion.Identifier(),
					Kind:       v2beta2.PolicyConfigKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "app-config-conflict",
				},
				Spec: v2beta2.PolicyConfigSpec{
					Match: v2beta2.PolicyConfigTarget{
						Applications: []v2beta2.PolicyTargetApplication{
							{
								Kind:      "HelmRelease",
								Name:      "helm-app",
								Namespace: "flux-system",
							},
						},
					},
					Config: map[string]v2beta2.PolicyConfigConfig{
						minimumReplicaCountPolicy: {
							Parameters: map[string]apiextensionsv1.JSON{},
						},
					},
				},
			},
			{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v2beta2.GroupVersion.Identifier(),
					Kind:       v2beta2.PolicyConfigKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "resource-config-conflict",
				},
				Spec: v2beta2.PolicyConfigSpec{
					Match: v2beta2.PolicyConfigTarget{
						Resources: []v2beta2.PolicyTargetResource{
							{
								Kind:      "Deployment",
								Name:      "test-deployment",
								Namespace: "default",
							},
						},
					},
					Config: map[string]v2beta2.PolicyConfigConfig{
						minimumReplicaCountPolicy: {
							Parameters: map[string]apiextensionsv1.JSON{},
						},
					},
				},
			},
		}
		for i := range configs {
			err := cl.Create(ctx, &configs[i])
			assert.NotNil(t, err, "expected an error when trying to create config %s", configs[i].Name)
		}
	})

	t.Run("test mutate resources", func(t *testing.T) {
		err := kubectl("apply", "-f", "data/resources/mutation_test-resources.yaml")
		assert.NotNil(t, err)

		var policy v2beta2.Policy
		err = cl.Get(ctx, client.ObjectKey{Name: minimumReplicaCountPolicy}, &policy)
		assert.Nil(t, err)

		policy.Spec.Mutate = true
		err = cl.Update(ctx, &policy)
		assert.Nil(t, err)

		err = kubectl("apply", "-f", "data/resources/mutation_test-resources.yaml")
		assert.Nil(t, err)

		var deployment appsv1.Deployment
		err = cl.Get(ctx, client.ObjectKey{Name: testMutationDeployment, Namespace: "default"}, &deployment)
		assert.Nil(t, err)

		assert.NotNil(t, deployment.Spec.Replicas)
		assert.Equal(t, int32(3), *deployment.Spec.Replicas)
	})
}
