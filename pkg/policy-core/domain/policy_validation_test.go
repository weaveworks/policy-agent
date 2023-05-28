package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/weaveworks/policy-agent/pkg/uuid-go"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestPolicyToEvent(t *testing.T) {
	policy := Policy{
		ID:       uuid.NewV4().String(),
		Name:     "my-policy",
		Category: "my-category",
		Severity: "low",
		Reference: v1.ObjectReference{
			UID:             "my-policy",
			APIVersion:      "pac.weave.works/v1",
			Kind:            "Policy",
			Name:            "my-policy",
			ResourceVersion: "1",
		},
		Tags: []string{"tag"},
		Standards: []PolicyStandard{
			{
				ID:       "stnd",
				Controls: []string{"1.1.1"},
			},
		},
		Parameters: []PolicyParameters{
			{
				Name:      "param1",
				Value:     "test",
				Type:      "string",
				Required:  true,
				ConfigRef: "config-1",
			},
		},
		Modes: []string{"audit", "admission"},
	}

	entity := Entity{
		ID:              uuid.NewV4().String(),
		APIVersion:      "v1",
		Kind:            "Deployment",
		Name:            "my-deployment",
		Namespace:       "default",
		Manifest:        map[string]interface{}{},
		ResourceVersion: "1",
		Labels:          map[string]string{},
	}

	results := []PolicyValidation{
		{
			Policy:    policy,
			Entity:    entity,
			Status:    PolicyValidationStatusViolating,
			Message:   "message",
			Type:      "Admission",
			Trigger:   "Admission",
			CreatedAt: time.Now(),
			Occurrences: []Occurrence{
				{
					Message: "test",
				},
			},
		},
		{
			Policy:    policy,
			Entity:    entity,
			Status:    PolicyValidationStatusCompliant,
			Message:   "message",
			Type:      "Audit",
			Trigger:   "PolicyChange",
			CreatedAt: time.Now(),
			Occurrences: []Occurrence{
				{
					Message: "test",
				},
			},
		},
	}

	for _, result := range results {
		event, err := NewK8sEventFromPolicyValidation(result)
		assert.Nil(t, err)

		if result.Status == PolicyValidationStatusViolating {
			assert.Equal(t, event.Type, v1.EventTypeWarning)
			assert.Equal(t, event.Reason, EventReasonPolicyViolation)
			assert.Equal(t, event.Action, EventActionRejected)

		} else if result.Status == PolicyValidationStatusCompliant {
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
		manifest, err := json.Marshal(result.Entity.Manifest)
		assert.Nil(t, err)
		standards, err := json.Marshal(result.Policy.Standards)
		assert.Nil(t, err)
		occurrences, err := json.Marshal(result.Occurrences)
		assert.Nil(t, err)
		parameters, err := json.Marshal(result.Policy.Parameters)
		assert.Nil(t, err)

		assert.Equal(t, event.Annotations, map[string]string{
			"account_id":      result.AccountID,
			"cluster_id":      result.ClusterID,
			"policy_id":       result.Policy.ID,
			"policy_name":     result.Policy.Name,
			"severity":        result.Policy.Severity,
			"category":        result.Policy.Category,
			"description":     result.Policy.Description,
			"how_to_solve":    result.Policy.HowToSolve,
			"tags":            strings.Join(result.Policy.Tags, ","),
			"standards":       string(standards),
			"entity_manifest": string(manifest),
			"occurrences":     string(occurrences),
			"parameters":      string(parameters),
			"modes":           "audit,admission",
		})
		assert.Equal(t, event.Labels, map[string]string{
			PolicyValidationIDLabel:      result.ID,
			PolicyValidationTypeLabel:    result.Type,
			PolicyValidationTriggerLabel: result.Trigger,
		})
	}
}

func TestEventToPolicy(t *testing.T) {
	event := v1.Event{
		InvolvedObject: v1.ObjectReference{
			APIVersion:      "v1",
			Kind:            "Deployment",
			UID:             types.UID(uuid.NewV4().String()),
			Name:            "my-deployment",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"account_id":      uuid.NewV4().String(),
				"cluster_id":      uuid.NewV4().String(),
				"policy_id":       uuid.NewV4().String(),
				"description":     uuid.NewV4().String(),
				"how_to_solve":    uuid.NewV4().String(),
				"policy_name":     "my-policy",
				"category":        "category",
				"severity":        "high",
				"tags":            "tag1,tag2",
				"entity_manifest": `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"nginx-deployment","namespace":"default","uid":"af912668-957b-46d4-bc7a-51e6994cba56"},"spec":{"template":{"spec":{"containers":[{"image":"nginx:latest","imagePullPolicy":"Always","name":"nginx","ports":[{"containerPort":80,"protocol":"TCP"}]}]}}}}`,
				"standards":       `[{"id":"weave.standards.cis-benchmark","controls":["weave.controls.cis-benchmark.5.5.1"]},{"id":"weave.standards.mitre-attack","controls":["weave.controls.mitre-attack.1.2"]},{"id":"weave.standards.gdpr","controls":["weave.controls.gdpr.25","weave.controls.gdpr.32","weave.controls.gdpr.24"]},{"id":"weave.standards.soc2-type-i","controls":["weave.controls.soc2-type-i.1.6.8"]}]`,
				"occurrences":     `[{"message":"test1"},{"message":"test2"}]`,
				"modes":           "audit,admission",
			},
			Labels: map[string]string{
				PolicyValidationIDLabel:      uuid.NewV4().String(),
				PolicyValidationTypeLabel:    "Admission",
				PolicyValidationTriggerLabel: "Admission",
			},
		},
		Message: "Policy event",
		Reason:  "PolicyViolation",
		Related: &v1.ObjectReference{
			UID:             "my-policy",
			APIVersion:      "pac.weave.works/v1",
			Kind:            "Policy",
			Name:            "my-policy",
			ResourceVersion: "1",
		},
	}

	policyValidation, err := NewPolicyValidationFRomK8sEvent(&event)
	assert.Nil(t, err)

	assert.Equal(t, policyValidation.Status, PolicyValidationStatusViolating)
	assert.Equal(t, event.InvolvedObject.APIVersion, policyValidation.Entity.APIVersion)
	assert.Equal(t, event.InvolvedObject.Kind, policyValidation.Entity.Kind)
	assert.Equal(t, event.InvolvedObject.Name, policyValidation.Entity.Name)
	assert.Equal(t, event.InvolvedObject.Namespace, policyValidation.Entity.Namespace)

	policyRef := policyValidation.Policy.Reference.(*v1.ObjectReference)
	assert.Equal(t, event.Related.APIVersion, policyRef.APIVersion)
	assert.Equal(t, event.Related.Kind, policyRef.Kind)
	assert.Equal(t, event.Related.Name, policyRef.Name)

	assert.Equal(t, event.Message, policyValidation.Message)

	// verify metadata
	manifest, err := json.Marshal(policyValidation.Entity.Manifest)
	assert.Nil(t, err)
	standards, err := json.Marshal(policyValidation.Policy.Standards)
	assert.Nil(t, err)
	occurrences, err := json.Marshal(policyValidation.Occurrences)
	assert.Nil(t, err)

	assert.Equal(t, []string{"audit", "admission"}, policyValidation.Policy.Modes)

	assert.Equal(t, event.Annotations, map[string]string{
		"account_id":      policyValidation.AccountID,
		"cluster_id":      policyValidation.ClusterID,
		"policy_id":       policyValidation.Policy.ID,
		"policy_name":     policyValidation.Policy.Name,
		"severity":        policyValidation.Policy.Severity,
		"category":        policyValidation.Policy.Category,
		"description":     policyValidation.Policy.Description,
		"how_to_solve":    policyValidation.Policy.HowToSolve,
		"tags":            strings.Join(policyValidation.Policy.Tags, ","),
		"standards":       string(standards),
		"occurrences":     string(occurrences),
		"entity_manifest": string(manifest),
		"modes":           "audit,admission",
	})

	assert.Equal(t, event.Labels, map[string]string{
		PolicyValidationIDLabel:      policyValidation.ID,
		PolicyValidationTypeLabel:    policyValidation.Type,
		PolicyValidationTriggerLabel: policyValidation.Trigger,
	})
}
