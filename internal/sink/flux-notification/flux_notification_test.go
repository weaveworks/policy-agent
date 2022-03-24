package flux_notification

import (
	"context"
	"testing"
	"time"

	"github.com/MagalixTechnologies/uuid-go"
	"github.com/stretchr/testify/assert"
	"github.com/weaveworks/magalix-policy-agent/pkg/domain"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

func TestFluxNotificationSink(t *testing.T) {
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
			APIVersion:      "magalix.com/v1",
			Kind:            "Policy",
			Name:            "my-policy",
			ResourceVersion: "1",
		},
	}

	helmReleaseEntity := domain.Entity{
		ID:              uuid.NewV4().String(),
		APIVersion:      "v1",
		Kind:            "Deployment",
		Name:            "my-entity",
		Namespace:       "default",
		Manifest:        map[string]interface{}{},
		ResourceVersion: "1",
		Labels: map[string]string{
			"helm.toolkit.fluxcd.io/name":      "my-helm-release",
			"helm.toolkit.fluxcd.io/namespace": "flux-system",
		},
	}
	kustomizationEntity := domain.Entity{
		ID:              uuid.NewV4().String(),
		APIVersion:      "v1",
		Kind:            "Deployment",
		Name:            "my-entity",
		Namespace:       "default",
		Manifest:        map[string]interface{}{},
		ResourceVersion: "1",
		Labels: map[string]string{
			"kustomize.toolkit.fluxcd.io/name":      "my-kustomization",
			"kustomize.toolkit.fluxcd.io/namespace": "flux-system",
		},
	}

	results := []domain.PolicyValidation{
		{
			ID:        uuid.NewV4().String(),
			Policy:    policy,
			Entity:    helmReleaseEntity,
			Status:    domain.PolicyValidationStatusViolating,
			Message:   "message",
			Type:      "Admission",
			Trigger:   "Admission",
			CreatedAt: time.Now(),
		},
		{
			ID:        uuid.NewV4().String(),
			Policy:    policy,
			Entity:    kustomizationEntity,
			Status:    domain.PolicyValidationStatusCompliant,
			Message:   "message",
			Type:      "Admission",
			Trigger:   "Admission",
			CreatedAt: time.Now(),
		},
	}

	recorder := record.NewFakeRecorder(10)
	sink, err := NewFluxNotificationSink(recorder, "", "", "")
	if err != nil {
		t.Error(err)
	}

	ctx := context.Background()
	sink.Start(ctx)

	err = sink.Write(ctx, results)
	if err != nil {
		t.Error(err)
	}

	time.Sleep(1 * time.Second)

	assert.Equal(t, len(recorder.Events), 2)
}
