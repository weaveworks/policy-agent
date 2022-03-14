package events

import (
	"fmt"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EventActionAllowed          = "Allowed"
	EventActionRejected         = "Rejected"
	EventReasonPolicyViolation  = "PolicyViolation"
	EventReasonPolicyCompliance = "PolicyCompliance"
)

// EventFromPolicyValidationResult gets kubernetes event object from policy violation result object
func EventFromPolicyValidationResult(result domain.PolicyValidation) v1.Event {
	var reason, action, etype string

	if result.Status == domain.PolicyValidationStatusViolating {
		etype = v1.EventTypeWarning
		reason = EventReasonPolicyViolation
		action = EventActionRejected
	} else {
		etype = v1.EventTypeNormal
		reason = EventReasonPolicyCompliance
		action = EventActionAllowed
	}

	annotations := map[string]string{
		"account_id": result.AccountID,
		"cluster_id": result.ClusterID,
		"id":         result.ID,
		"policy":     result.Policy.ID,
		"severity":   result.Policy.Severity,
		"category":   result.Policy.Category,
		"type":       result.Type,
		"trigger":    result.Trigger,
	}

	namespace := result.Entity.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	involvedObject := result.Entity.ObjectRef()
	relatedObject := result.Policy.ObjectRef()

	timestamp := metav1.Time{}

	event := v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%v.%x", result.Entity.Name, timestamp.UnixNano()),
			Namespace:   namespace,
			Annotations: annotations,
		},
		InvolvedObject: *involvedObject,
		Related:        relatedObject,
		Type:           etype,
		Reason:         reason,
		Action:         action,
		Message:        result.Message,
		FirstTimestamp: timestamp,
		LastTimestamp:  timestamp,
	}

	return event
}
