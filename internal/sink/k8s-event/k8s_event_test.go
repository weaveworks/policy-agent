package k8s_event

import (
	"context"
	"testing"
	"time"

	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/MagalixTechnologies/uuid-go"
	"github.com/stretchr/testify/assert"
	mglx_events "github.com/weaveworks/policy-agent/pkg/events"
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

	results := []domain.PolicyValidation{
		{
			ID:        uuid.NewV4().String(),
			Policy:    policy,
			Entity:    violatingEntity,
			Status:    domain.PolicyValidationStatusViolating,
			Message:   "message",
			Type:      "Admission",
			Trigger:   "Admission",
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.NewV4().String(),
			Policy:    policy,
			Entity:    compliantEntity,
			Status:    domain.PolicyValidationStatusCompliant,
			Message:   "message",
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
	sink.Start(ctx)

	err = sink.Write(ctx, results)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(2 * time.Second)

	events, err := sink.kubeClient.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, len(events.Items), 2, "did not receive expected events")

	for _, event := range events.Items {
		if event.Type == v1.EventTypeWarning {
			assert.Equal(t, event.Reason, mglx_events.EventReasonPolicyViolation)
			assert.Equal(t, event.Action, mglx_events.EventActionRejected)

			// verify involved object holds entity info
			assert.Equal(t, event.InvolvedObject.APIVersion, violatingEntity.APIVersion)
			assert.Equal(t, event.InvolvedObject.Kind, violatingEntity.Kind)
			assert.Equal(t, event.InvolvedObject.Name, violatingEntity.Name)
			assert.Equal(t, event.InvolvedObject.Namespace, violatingEntity.Namespace)

		} else if event.Type == v1.EventTypeNormal {
			assert.Equal(t, event.Reason, mglx_events.EventReasonPolicyCompliance)
			assert.Equal(t, event.Action, mglx_events.EventActionAllowed)

			// verify involved object holds entity info
			assert.Equal(t, event.InvolvedObject.APIVersion, compliantEntity.APIVersion)
			assert.Equal(t, event.InvolvedObject.Kind, compliantEntity.Kind)
			assert.Equal(t, event.InvolvedObject.Name, compliantEntity.Name)
			assert.Equal(t, event.InvolvedObject.Namespace, compliantEntity.Namespace)
		}

		// verify involved object holds entity info
		policyRef := policy.Reference.(v1.ObjectReference)
		assert.Equal(t, event.Related.APIVersion, policyRef.APIVersion)
		assert.Equal(t, event.Related.Kind, policyRef.Kind)
		assert.Equal(t, event.Related.Name, policyRef.Name)
	}
}
