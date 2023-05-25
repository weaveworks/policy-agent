package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PolicyValidationStatusViolating = "Violation"
	PolicyValidationStatusCompliant = "Compliance"
	EventActionAllowed              = "Allowed"
	EventActionRejected             = "Rejected"
	EventReasonPolicyViolation      = "PolicyViolation"
	EventReasonPolicyCompliance     = "PolicyCompliance"
	PolicyValidationTypeLabel       = "pac.weave.works/type"
	PolicyValidationIDLabel         = "pac.weave.works/id"
	PolicyValidationTriggerLabel    = "pac.weave.works/trigger"
)

// IaCMetadata defines the values of type iac for validation
type IaCMetadata struct {
	Branch        string                 `json:"branch" validate:"required"`
	Commit        string                 `json:"commit" validate:"required"`
	File          string                 `json:"file" validate:"required"`
	PlatformName  string                 `json:"platform_name"`
	PlatformInfo  map[string]interface{} `json:"platform"`
	Repository    string                 `json:"repository" validate:"required"`
	ResultUrl     string                 `json:"result_url"`
	Source        string                 `json:"source" validate:"required"`
	Type          string                 `json:"type" validate:"oneof=IaC Generic"`
	KubeGuardID   string                 `json:"kubeguard_id"`
	KubeGuardName string                 `json:"kubeguard_name"`
	Provider      string                 `json:"provider"`
	PullRequest   string                 `json:"pull_request"`
}

type Occurrence struct {
	Message          string      `json:"message"`
	ViolatingKey     *string     `json:"violating_key,omitempty"`
	RecommendedValue interface{} `json:"recommended_value,omitempty"`
	Mutated          bool        `json:"-"`
}

// PolicyValidation defines the result of a policy validation result against an entity
type PolicyValidation struct {
	ID          string       `json:"id"`
	AccountID   string       `json:"account_id"`
	ClusterID   string       `json:"cluster_id"`
	Policy      Policy       `json:"policy"`
	Entity      Entity       `json:"entity"`
	Status      string       `json:"status"`
	Message     string       `json:"message"`
	Occurrences []Occurrence `json:"occurrences"`
	Type        string       `json:"source"`
	Trigger     string       `json:"trigger"`
	CreatedAt   time.Time    `json:"created_at"`
	Metadata    interface{}  `json:"metadata"`
}

// PolicyValidationSummary contains violation and compliance result of a validate operation
type PolicyValidationSummary struct {
	Violations  []PolicyValidation
	Compliances []PolicyValidation
	Mutation    *MutationResult
}

// GetViolationMessages get all violation messages from review results
func (v *PolicyValidationSummary) GetViolationMessages() []string {
	var messages []string
	for _, violation := range v.Violations {
		messages = append(messages, violation.Message)
	}
	return messages
}

// GetViolationOccurrencesMessages get all occurrences messages from review results
func (v *PolicyValidationSummary) GetViolationOccurrencesMessages() []string {
	var messages []string
	for _, violation := range v.Violations {
		for _, occurrence := range violation.Occurrences {
			messages = append(messages, occurrence.Message)
		}
	}
	return messages
}

// NewK8sEventFromPolicyVlidation gets kubernetes event object from policy violation result object
func NewK8sEventFromPolicyValidation(result PolicyValidation) (*v1.Event, error) {
	var reason, action, etype string

	if result.Status == PolicyValidationStatusViolating {
		etype = v1.EventTypeWarning
		reason = EventReasonPolicyViolation
		action = EventActionRejected
	} else {
		etype = v1.EventTypeNormal
		reason = EventReasonPolicyCompliance
		action = EventActionAllowed
	}

	standards, err := json.Marshal(result.Policy.Standards)
	if err != nil {
		return nil, fmt.Errorf("failed to parse policy validation standards: %w", err)
	}
	manifest, err := json.Marshal(result.Entity.Manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse policy validation entity manifest: %w", err)
	}

	occurrences, err := json.Marshal(result.Occurrences)
	if err != nil {
		return nil, fmt.Errorf("failed to parse policy validation occurrences: %w", err)
	}

	tags := strings.Join(result.Policy.Tags, ",")

	parameters, err := json.Marshal(result.Policy.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to parse policy validation parameters config: %w", err)
	}

	annotations := map[string]string{
		"account_id":      result.AccountID,
		"cluster_id":      result.ClusterID,
		"policy_id":       result.Policy.ID,
		"policy_name":     result.Policy.Name,
		"severity":        result.Policy.Severity,
		"category":        result.Policy.Category,
		"standards":       string(standards),
		"entity_manifest": string(manifest),
		"occurrences":     string(occurrences),
		"tags":            tags,
		"description":     result.Policy.Description,
		"how_to_solve":    result.Policy.HowToSolve,
		"parameters":      string(parameters),
		"modes":           strings.Join(result.Policy.Modes, ","),
	}

	namespace := result.Entity.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	involvedObject := result.Entity.ObjectRef()
	relatedObject := result.Policy.ObjectRef()

	timestamp := metav1.NewTime(time.Now())

	event := &v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%v.%x", result.Entity.Name, timestamp.UnixNano()),
			Namespace:   namespace,
			Annotations: annotations,
			Labels: map[string]string{
				PolicyValidationIDLabel:      result.ID,
				PolicyValidationTypeLabel:    result.Type,
				PolicyValidationTriggerLabel: result.Trigger,
			},
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

	return event, nil
}

// NewPolicyValidationFRomK8sEvent gets policy violation result object from kubernetes event object
func NewPolicyValidationFRomK8sEvent(event *v1.Event) (PolicyValidation, error) {
	labels := event.ObjectMeta.Labels
	annotations := event.ObjectMeta.Annotations

	var status string
	if event.Reason == EventReasonPolicyViolation {
		status = PolicyValidationStatusViolating
	} else {
		status = PolicyValidationStatusCompliant
	}
	policyValidation := PolicyValidation{
		AccountID: annotations["account_id"],
		ClusterID: annotations["cluster_id"],
		ID:        labels[PolicyValidationIDLabel],
		Type:      labels[PolicyValidationTypeLabel],
		Trigger:   labels[PolicyValidationTriggerLabel],
		CreatedAt: event.FirstTimestamp.Time,
		Message:   event.Message,
		Status:    status,
		Policy: Policy{
			ID:          annotations["policy_id"],
			Name:        annotations["policy_name"],
			Category:    annotations["category"],
			Severity:    annotations["severity"],
			Description: annotations["description"],
			HowToSolve:  annotations["how_to_solve"],
			Reference:   event.Related,
			Tags:        strings.Split(annotations["tags"], ","),
			Modes:       strings.Split(annotations["modes"], ","),
		},
		Entity: Entity{
			APIVersion:      event.InvolvedObject.APIVersion,
			Kind:            event.InvolvedObject.Kind,
			ID:              string(event.InvolvedObject.UID),
			Name:            event.InvolvedObject.Name,
			Namespace:       event.InvolvedObject.Namespace,
			ResourceVersion: event.InvolvedObject.ResourceVersion,
		},
	}
	err := json.Unmarshal([]byte(annotations["standards"]), &policyValidation.Policy.Standards)
	if err != nil {
		return policyValidation, fmt.Errorf("failed to get standards from event: %w", err)
	}
	err = json.Unmarshal([]byte(annotations["entity_manifest"]), &policyValidation.Entity.Manifest)
	if err != nil {
		return policyValidation, fmt.Errorf("failed to get entity manifest from event: %w", err)
	}
	err = json.Unmarshal([]byte(annotations["occurrences"]), &policyValidation.Occurrences)
	if err != nil {
		return policyValidation, fmt.Errorf("failed to get occurrences from event: %w", err)
	}
	if _, ok := annotations["parameters"]; ok {
		err = json.Unmarshal([]byte(annotations["parameters"]), &policyValidation.Policy.Parameters)
		if err != nil {
			return policyValidation, fmt.Errorf("failed to get policy parameters from event: %w", err)
		}
	}

	return policyValidation, nil
}
