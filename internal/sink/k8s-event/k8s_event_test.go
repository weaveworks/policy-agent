package k8s_event

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/weaveworks/weave-policy-agent/pkg/policy-core/domain"
	"github.com/weaveworks/weave-policy-agent/pkg/uuid-go"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sEventSink(t *testing.T) {
	// test policy
	policy := domain.Policy{
		ID:          uuid.NewV4().String(),
		Name:        "policy",
		Code:        "code",
		Description: "description",
		HowToSolve:  "how_to_solve",
		Category:    "category",
		Severity:    "severity",
		Reference: v1.ObjectReference{
			UID:             types.UID(uuid.NewV4().String()),
			APIVersion:      "pac.weave.works/v1",
			Kind:            "Policy",
			Name:            "my-policy",
			ResourceVersion: "1",
		},
	}

	violatingEntity := domain.Entity{
		ID:              uuid.NewV4().String(),
		APIVersion:      "v1",
		Kind:            "Deployment",
		Name:            "my-violating-entity",
		Namespace:       "default",
		Manifest:        map[string]interface{}{},
		ResourceVersion: "1",
		Labels:          map[string]string{},
	}

	compliantEntity := domain.Entity{
		ID:              uuid.NewV4().String(),
		APIVersion:      "v1",
		Kind:            "Deployment",
		Name:            "my-compliant-entity",
		Namespace:       "default",
		Manifest:        map[string]interface{}{},
		ResourceVersion: "1",
		Labels:          map[string]string{},
	}

	fluxHelmViolatingEntity := domain.Entity{
		ID:              uuid.NewV4().String(),
		APIVersion:      "v1",
		Kind:            "Deployment",
		Name:            "my-helm-violating-entity",
		Namespace:       "default",
		Manifest:        map[string]interface{}{},
		ResourceVersion: "1",
		Labels: map[string]string{
			"helm.toolkit.fluxcd.io/name":      "my-helm-app-name",
			"helm.toolkit.fluxcd.io/namespace": "my-helm-app-namespace",
		},
	}

	fluxKustomizeViolatingEntity := domain.Entity{
		ID:              uuid.NewV4().String(),
		APIVersion:      "v1",
		Kind:            "Deployment",
		Name:            "my-kustomize-violating-entity",
		Namespace:       "default",
		Manifest:        map[string]interface{}{},
		ResourceVersion: "1",
		Labels: map[string]string{
			"kustomize.toolkit.fluxcd.io/name":      "my-kustomize-app-name",
			"kustomize.toolkit.fluxcd.io/namespace": "my-kustomize-app-namespace",
		},
	}

	results := []domain.PolicyValidation{
		{
			ID:        uuid.NewV4().String(),
			Policy:    policy,
			Entity:    violatingEntity,
			Status:    domain.PolicyValidationStatusViolating,
			Message:   "violating-entity",
			Type:      "Admission",
			Trigger:   "Admission",
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.NewV4().String(),
			Policy:    policy,
			Entity:    compliantEntity,
			Status:    domain.PolicyValidationStatusCompliant,
			Message:   "compliant-entity",
			Type:      "Admission",
			Trigger:   "Admission",
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.NewV4().String(),
			Policy:    policy,
			Entity:    fluxHelmViolatingEntity,
			Status:    domain.PolicyValidationStatusViolating,
			Message:   "flux-helm-entity",
			Type:      "Admission",
			Trigger:   "Admission",
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.NewV4().String(),
			Policy:    policy,
			Entity:    fluxKustomizeViolatingEntity,
			Status:    domain.PolicyValidationStatusViolating,
			Message:   "flux-kustomize-entity",
			Type:      "Admission",
			Trigger:   "Admission",
			CreatedAt: time.Now(),
		},
	}

	sink, err := NewK8sEventSink(fake.NewSimpleClientset(), "", "", "policy-agent")
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	go sink.Start(ctx)

	err = sink.Write(ctx, results)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(4 * time.Second)

	events, err := sink.kubeClient.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(events.Items), 4, "did not receive expected events")

	for _, event := range events.Items {
		if event.Message == "violating-entity" {
			assert.Equal(t, event.Reason, domain.EventReasonPolicyViolation)
			assert.Equal(t, event.Action, domain.EventActionRejected)
			// verify involved object holds entity info
			assert.Equal(t, event.InvolvedObject.APIVersion, violatingEntity.APIVersion)
			assert.Equal(t, event.InvolvedObject.Kind, violatingEntity.Kind)
			assert.Equal(t, event.InvolvedObject.Name, violatingEntity.Name)
			assert.Equal(t, event.InvolvedObject.Namespace, violatingEntity.Namespace)

		} else if event.Message == "compliant-entity" {
			assert.Equal(t, event.Reason, domain.EventReasonPolicyCompliance)
			assert.Equal(t, event.Action, domain.EventActionAllowed)
			// verify involved object holds entity info
			assert.Equal(t, event.InvolvedObject.APIVersion, compliantEntity.APIVersion)
			assert.Equal(t, event.InvolvedObject.Kind, compliantEntity.Kind)
			assert.Equal(t, event.InvolvedObject.Name, compliantEntity.Name)
			assert.Equal(t, event.InvolvedObject.Namespace, compliantEntity.Namespace)
		} else if event.Message == "flux-helm-entity" {
			assert.Equal(t, event.Reason, domain.EventReasonPolicyViolation)
			assert.Equal(t, event.Action, domain.EventActionRejected)
			// verify involved object holds entity info
			assert.Equal(t, event.InvolvedObject.APIVersion, "helm.toolkit.fluxcd.io")
			assert.Equal(t, event.InvolvedObject.Kind, "HelmRelease")
			assert.Equal(t, event.InvolvedObject.Name, "my-helm-app-name")
			assert.Equal(t, event.InvolvedObject.Namespace, "my-helm-app-namespace")
		} else if event.Message == "compliant-entity" {
			assert.Equal(t, event.Reason, domain.EventReasonPolicyViolation)
			assert.Equal(t, event.Action, domain.EventActionRejected)
			// verify involved object holds entity info
			assert.Equal(t, event.InvolvedObject.APIVersion, "kustomize.toolkit.fluxcd.io")
			assert.Equal(t, event.InvolvedObject.Kind, "Kustomization")
			assert.Equal(t, event.InvolvedObject.Name, "my-kustomize-app-name")
			assert.Equal(t, event.InvolvedObject.Namespace, "my-kustomize-app-namespace")
		}

		// verify involved object holds entity info
		policyRef := policy.Reference.(v1.ObjectReference)
		assert.Equal(t, event.Related.APIVersion, policyRef.APIVersion)
		assert.Equal(t, event.Related.Kind, policyRef.Kind)
		assert.Equal(t, event.Related.Name, policyRef.Name)
	}
}
