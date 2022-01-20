package domain

import (
	"time"
)

const (
	ValidationResultStatusViolating = "Violation"
	ValidationResultStatusCompliant = "Compliance"
)

type ValidationResult struct {
	//@TODO ACCount id cluster id
	ID        string    `json:"id"`
	Policy    Policy    `json:"policy"`
	Entity    Entity    `json:"entity"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

type ValidationSummary struct {
	Violations  []ValidationResult
	Compliances []ValidationResult
}

// GetViolationMessages get all violation messages from review results
func (v *ValidationSummary) GetViolationMessages() []string {
	var messages []string
	for _, violation := range v.Violations {
		messages = append(messages, violation.Message)
	}
	return messages
}
