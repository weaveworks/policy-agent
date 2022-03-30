package events

import (
	"testing"
	"time"

	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/MagalixTechnologies/uuid-go"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestEvents(t *testing.T) {
	policy := domain.Policy{
		ID:       uuid.NewV4().String(),
		Name:     "my-policy",
		Category: "my-category",
		Severity: "low",
		Reference: v1.ObjectReference{
			UID:             "my-policy",
			APIVersion:      "magalix.com/v1",
			Kind:            "Policy",
			Name:            "my-policy",
			ResourceVersion: "1",
		},
	}

	entity := domain.Entity{
		ID:              uuid.NewV4().String(),
		APIVersion:      "v1",
		Kind:            "Deployment",
		Name:            "my-deployment",
		Namespace:       "default",
		Manifest:        map[string]interface{}{},
		ResourceVersion: "1",
		Labels:          map[string]string{},
	}

	results := []domain.PolicyValidation{
		{
			Policy:    policy,
			Entity:    entity,
			Status:    domain.PolicyValidationStatusViolating,
			Message:   "message",
			Type:      "Admission",
			Trigger:   "Admission",
			CreatedAt: time.Now(),
		},
		{
			Policy:    policy,
			Entity:    entity,
			Status:    domain.PolicyValidationStatusCompliant,
			Message:   "message",
			Type:      "Audit",
			Trigger:   "PolicyChange",
			CreatedAt: time.Now(),
		},
	}

	for _, result := range results {
		event := EventFromPolicyValidationResult(result)

		if result.Status == domain.PolicyValidationStatusViolating {
			assert.Equal(t, event.Type, v1.EventTypeWarning)
			assert.Equal(t, event.Reason, EventReasonPolicyViolation)
			assert.Equal(t, event.Action, EventActionRejected)

		} else if result.Status == domain.PolicyValidationStatusCompliant {
			assert.Equal(t, event.Type, v1.EventTypeNormal)
			assert.Equal(t, event.Reason, EventReasonPolicyCompliance)
			assert.Equal(t, event.Action, EventActionAllowed)
		}

		// verify involved object holds entity info
		assert.Equal(t, event.InvolvedObject.APIVersion, entity.APIVersion)
		assert.Equal(t, event.InvolvedObject.Kind, entity.Kind)
		assert.Equal(t, event.InvolvedObject.Name, entity.Name)
		assert.Equal(t, event.InvolvedObject.Namespace, entity.Namespace)

		// verify involved object holds entity info
		policyRef := policy.Reference.(v1.ObjectReference)
		assert.Equal(t, event.Related.APIVersion, policyRef.APIVersion)
		assert.Equal(t, event.Related.Kind, policyRef.Kind)
		assert.Equal(t, event.Related.Name, policyRef.Name)

		// verify event message
		assert.Equal(t, event.Message, result.Message)

		// verify metadata
		assert.Equal(t, event.Annotations, map[string]string{
			"account_id": result.AccountID,
			"cluster_id": result.ClusterID,
			"id":         result.ID,
			"policy":     result.Policy.ID,
			"severity":   result.Policy.Severity,
			"category":   result.Policy.Category,
			"type":       result.Type,
			"trigger":    result.Trigger,
		})
	}
}
