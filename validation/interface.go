package validation

import "context"

type PoliciesSource interface {
	GetPolicies(ctx context.Context) ([]Policy, error)
}

type ValidationResultSink interface {
	Write(ctx context.Context, violations []ValidationResult) error
}
