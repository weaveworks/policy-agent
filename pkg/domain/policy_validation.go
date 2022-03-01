package domain

import (
	"time"
)

const (
	PolicyValidationStatusViolating = "Violation"
	PolicyValidationStatusCompliant = "Compliance"
)

// PolicyValidation defines the result of a policy validation result against an entity
type PolicyValidation struct {
	ID        string    `json:"id"`
	AccountID string    `json:"account_id"`
	ClusterID string    `json:"cluster_id"`
	Policy    Policy    `json:"policy"`
	Entity    Entity    `json:"entity"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Type      string    `json:"source"`
	Trigger   string    `json:"trigger"`
	CreatedAt time.Time `json:"created_at"`
}

// PolicyValidationSummary contains violation and compliance result of a validate operation
type PolicyValidationSummary struct {
	Violations  []PolicyValidation
	Compliances []PolicyValidation
}

// GetViolationMessages get all violation messages from review results
func (v *PolicyValidationSummary) GetViolationMessages() []string {
	var messages []string
	for _, violation := range v.Violations {
		messages = append(messages, violation.Message)
	}
	return messages
}
