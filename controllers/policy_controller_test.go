package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestController(t *testing.T) {
	testEnv := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
		},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err = testEnv.Stop()
		if err != nil {
			t.Error(err)
		}
	}()

	if err := pacv2.AddToScheme(scheme.Scheme); err != nil {
		t.Fatal(err)
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatal(err)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatal(err)
	}

	reconciler := &PolicyReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
	}

	if err := reconciler.SetupWithManager(mgr); err != nil {
		t.Fatal(err)
	}

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
		{
			id:        "policy-3",
			category:  "category-x",
			severity:  "low",
			provider:  pacv2.PolicyTerraformProvider,
			standards: []string{"standard-x"},
			tags:      []string{"tag-x"},
		},
		{
			id:        "policy-4",
			category:  "category-y",
			severity:  "high",
			provider:  pacv2.PolicyTerraformProvider,
			standards: []string{"standard-y"},
			tags:      []string{"tag-y"},
		},
	}

	ctx := context.TODO()
	for _, item := range policyData {
		policy := createPolicy(
			item.id,
			item.category,
			item.severity,
			item.provider,
			item.standards,
			item.tags,
		)
		if err = k8sClient.Create(ctx, &policy); err != nil {
			t.Fatal(err)
		}
		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(&policy)})
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("check policies modes updated", func(t *testing.T) {
		expectedModes := map[string][]string{
			"policy-1": {"audit", "admission"},
			"policy-2": {"audit", "admission"},
			"policy-3": {"tf-admission"},
			"policy-4": {"tf-admission"},
		}
		var policies pacv2.PolicyList
		if err = k8sClient.List(context.Background(), &policies); err != nil {
			t.Fatal(err)
		}
		for _, policy := range policies.Items {
			assert.ElementsMatch(t, expectedModes[policy.Name], policy.Status.Modes)
			expectedLabels := map[string]string{}
			for _, mode := range expectedModes[policy.Name] {
				expectedLabels[fmt.Sprintf("%s.%s", pacv2.PolicyModeLabelPrefix, mode)] = ""
			}
			assert.EqualValues(t, expectedLabels, policy.Labels)
		}
	})

	t.Run("add new audit policy set 'policyset-1' to include policy-1 (policy-2 will be excluded)", func(t *testing.T) {
		expectedModes := map[string][]string{
			"policy-1": {"audit", "admission"},
			"policy-2": {"admission"},
			"policy-3": {"tf-admission"},
			"policy-4": {"tf-admission"},
		}

		policySet := createPolicySet("policyset-1", pacv2.PolicySetAuditMode, []string{"policy-1"}, nil, nil, nil, nil)
		if err = k8sClient.Create(ctx, &policySet); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(reconciler.reconcile(&policySet)), len(expectedModes))

		for policyID := range expectedModes {
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: policyID}})
			if err != nil {
				t.Fatal(err)
			}
		}

		var policies pacv2.PolicyList
		if err = k8sClient.List(context.Background(), &policies); err != nil {
			t.Fatal(err)
		}
		for _, policy := range policies.Items {
			assert.ElementsMatch(t, expectedModes[policy.Name], policy.Status.Modes)
			expectedLabels := map[string]string{}
			for _, mode := range expectedModes[policy.Name] {
				expectedLabels[fmt.Sprintf("%s.%s", pacv2.PolicyModeLabelPrefix, mode)] = ""
			}
			assert.EqualValues(t, expectedLabels, policy.Labels)
		}
	})

	t.Run("delete policyset 'policyset-1', policy-2 should be included in audit mode", func(t *testing.T) {
		expectedModes := map[string][]string{
			"policy-1": {"audit", "admission"},
			"policy-2": {"audit", "admission"},
			"policy-3": {"tf-admission"},
			"policy-4": {"tf-admission"},
		}

		policySet := pacv2.PolicySet{}
		policySet.SetName("policyset-1")
		if err = k8sClient.Delete(ctx, &policySet); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(reconciler.reconcile(&policySet)), len(expectedModes))

		for policyID := range expectedModes {
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: policyID}})
			if err != nil {
				t.Fatal(err)
			}
		}

		var policies pacv2.PolicyList
		if err = k8sClient.List(context.Background(), &policies); err != nil {
			t.Fatal(err)
		}
		for _, policy := range policies.Items {
			assert.ElementsMatch(t, expectedModes[policy.Name], policy.Status.Modes)
			expectedLabels := map[string]string{}
			for _, mode := range expectedModes[policy.Name] {
				expectedLabels[fmt.Sprintf("%s.%s", pacv2.PolicyModeLabelPrefix, mode)] = ""
			}
			assert.EqualValues(t, expectedLabels, policy.Labels)
		}
	})

	t.Run("add new admission policy set 'policyset-2' to include policy-1 (policy-2 will be excluded)", func(t *testing.T) {
		expectedModes := map[string][]string{
			"policy-1": {"audit", "admission"},
			"policy-2": {"audit"},
			"policy-3": {"tf-admission"},
			"policy-4": {"tf-admission"},
		}

		policySet := createPolicySet("policyset-2", pacv2.PolicySetAdmissionMode, []string{"policy-1"}, nil, nil, nil, nil)
		if err = k8sClient.Create(ctx, &policySet); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(reconciler.reconcile(&policySet)), len(expectedModes))

		for policyID := range expectedModes {
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: policyID}})
			if err != nil {
				t.Fatal(err)
			}
		}

		var policies pacv2.PolicyList
		if err = k8sClient.List(context.Background(), &policies); err != nil {
			t.Fatal(err)
		}
		for _, policy := range policies.Items {
			assert.ElementsMatch(t, expectedModes[policy.Name], policy.Status.Modes)
			expectedLabels := map[string]string{}
			for _, mode := range expectedModes[policy.Name] {
				expectedLabels[fmt.Sprintf("%s.%s", pacv2.PolicyModeLabelPrefix, mode)] = ""
			}
			assert.EqualValues(t, expectedLabels, policy.Labels)
		}
	})

	t.Run("delete policyset 'policyset-2', policy-2 should be included in admission mode", func(t *testing.T) {
		expectedModes := map[string][]string{
			"policy-1": {"audit", "admission"},
			"policy-2": {"audit", "admission"},
			"policy-3": {"tf-admission"},
			"policy-4": {"tf-admission"},
		}

		policySet := pacv2.PolicySet{}
		policySet.SetName("policyset-2")
		if err = k8sClient.Delete(ctx, &policySet); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(reconciler.reconcile(&policySet)), len(expectedModes))

		for policyID := range expectedModes {
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: policyID}})
			if err != nil {
				t.Fatal(err)
			}
		}

		var policies pacv2.PolicyList
		if err = k8sClient.List(context.Background(), &policies); err != nil {
			t.Fatal(err)
		}
		for _, policy := range policies.Items {
			assert.ElementsMatch(t, expectedModes[policy.Name], policy.Status.Modes)
			expectedLabels := map[string]string{}
			for _, mode := range expectedModes[policy.Name] {
				expectedLabels[fmt.Sprintf("%s.%s", pacv2.PolicyModeLabelPrefix, mode)] = ""
			}
			assert.EqualValues(t, expectedLabels, policy.Labels)
		}
	})

	t.Run("add new tf-admission policy set 'policyset-3' to include policy-3 (policy-4 will be excluded)", func(t *testing.T) {
		expectedModes := map[string][]string{
			"policy-1": {"audit", "admission"},
			"policy-2": {"audit", "admission"},
			"policy-3": {"tf-admission"},
			"policy-4": {},
		}

		policySet := createPolicySet("policyset-3", pacv2.PolicySetTFAdmissionMode, []string{"policy-3"}, nil, nil, nil, nil)
		if err = k8sClient.Create(ctx, &policySet); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(reconciler.reconcile(&policySet)), len(expectedModes))

		for policyID := range expectedModes {
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: policyID}})
			if err != nil {
				t.Fatal(err)
			}
		}

		var policies pacv2.PolicyList
		if err = k8sClient.List(context.Background(), &policies); err != nil {
			t.Fatal(err)
		}
		for _, policy := range policies.Items {
			assert.ElementsMatch(t, expectedModes[policy.Name], policy.Status.Modes)
			expectedLabels := map[string]string{}
			for _, mode := range expectedModes[policy.Name] {
				expectedLabels[fmt.Sprintf("%s.%s", pacv2.PolicyModeLabelPrefix, mode)] = ""
			}
			if len(expectedLabels) == 0 {
				expectedLabels = nil
			}
			assert.EqualValues(t, expectedLabels, policy.Labels)
		}
	})

	t.Run("delete policyset 'policyset-3', , policy-4 should be included in tf-admission mode", func(t *testing.T) {
		expectedModes := map[string][]string{
			"policy-1": {"audit", "admission"},
			"policy-2": {"audit", "admission"},
			"policy-3": {"tf-admission"},
			"policy-4": {"tf-admission"},
		}

		policySet := pacv2.PolicySet{}
		policySet.SetName("policyset-3")
		if err = k8sClient.Delete(ctx, &policySet); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, len(reconciler.reconcile(&policySet)), len(expectedModes))

		for policyID := range expectedModes {
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: policyID}})
			if err != nil {
				t.Fatal(err)
			}
		}

		var policies pacv2.PolicyList
		if err = k8sClient.List(context.Background(), &policies); err != nil {
			t.Fatal(err)
		}
		for _, policy := range policies.Items {
			assert.ElementsMatch(t, expectedModes[policy.Name], policy.Status.Modes)
			expectedLabels := map[string]string{}
			for _, mode := range expectedModes[policy.Name] {
				expectedLabels[fmt.Sprintf("%s.%s", pacv2.PolicyModeLabelPrefix, mode)] = ""
			}
			assert.EqualValues(t, expectedLabels, policy.Labels)
		}
	})
}

func createPolicy(id, category, severity, provider string, standards, tags []string) pacv2.Policy {
	policy := pacv2.Policy{
		TypeMeta: v1.TypeMeta{
			APIVersion: pacv2.GroupVersion.Identifier(),
			Kind:       pacv2.PolicyKind,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: id,
		},
		Spec: pacv2.PolicySpec{
			ID:       id,
			Name:     id,
			Category: category,
			Severity: severity,
			Provider: provider,
			Tags:     tags,
			Targets: pacv2.PolicyTargets{
				Kinds: []string{"Deployment"},
			},
		},
	}
	for i := range standards {
		policy.Spec.Standards = append(policy.Spec.Standards, pacv2.PolicyStandard{
			ID: standards[i],
		})
	}
	return policy
}

func createPolicySet(id, mode string, ids, categories, severities, standards, tags []string) pacv2.PolicySet {
	return pacv2.PolicySet{
		TypeMeta: v1.TypeMeta{
			APIVersion: pacv2.GroupVersion.Identifier(),
			Kind:       pacv2.PolicySetKind,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: id,
		},
		Spec: pacv2.PolicySetSpec{
			Mode: mode,
			Filters: pacv2.PolicySetFilters{
				IDs:        ids,
				Categories: categories,
				Severities: severities,
				Standards:  standards,
				Tags:       tags,
			},
		},
	}
}
