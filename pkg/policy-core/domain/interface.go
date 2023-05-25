package domain

import "context"

// PoliciesSource acts as a source for policies
type PoliciesSource interface {
	// GetAll returns all available policies
	GetAll(ctx context.Context) ([]Policy, error)
	GetPolicyConfig(ctx context.Context, entity Entity) (*PolicyConfig, error)
}

// PolicyValidationSink acts as a sink to send the results of a validation to
type PolicyValidationSink interface {
	// Write saves the results
	Write(ctx context.Context, PolicyValidations []PolicyValidation) error
}
