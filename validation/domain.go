package validation

import (
	"fmt"
	"time"
)

const (
	ValidationResultStatusViolating = "Violation"
	ValidationResultStatusCompliant = "Compliance"
)

type PolicyTargets struct {
	Kind      []string            `json:"kind"`
	Label     []map[string]string `json:"label"`
	Namespace []string            `json:"namespace"`
}

type PolicyParameters struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Required bool   `json:"required"`
}

type Policy struct {
	Name        string             `json:"name"`
	ID          string             `json:"id"`
	Code        string             `json:"code"`
	Parameters  []PolicyParameters `json:"parameters"`
	Targets     PolicyTargets      `json:"targets"`
	Description string             `json:"description"`
	HowToSolve  string             `json:"how_to_solve"`
	Category    string             `json:"category"`
	Tags        []string           `json:"tags"`
	Severity    string             `json:"severity"`
}

type ValidationResult struct {
	ID              string    `json:"id"`
	PolicyID        string    `json:"policy_id"`
	EntityName      string    `json:"entity_name"`
	EntityKind      string    `json:"entity_type"`
	Status          string    `json:"status"`
	Message         string    `json:"message"`
	Source          string    `json:"source"`
	CreatedAt       time.Time `json:"created_at"`
	EntityNamespace string    `json:"entity_namespace"`
	EntitySpec      string    `json:"entity_spec"`
}

type ValidationSummary struct {
	Violations  []ValidationResult
	Compliances []ValidationResult
}

// GetViolationMessages get all violation messages from review results
func (v *ValidationSummary) GetViolationMessages() []string {
	var messages []string
	for _, violation := range v.Violations {
		message := fmt.Sprintf("%s. Policy: %s", violation.Message, violation.PolicyID)
		messages = append(messages, message)
	}
	return messages
}
