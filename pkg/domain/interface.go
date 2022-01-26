package domain

import "context"

type PoliciesSource interface {
	GetAll(ctx context.Context) ([]Policy, error)
}

type ValidationResultSink interface {
	Write(ctx context.Context, validationResults []ValidationResult) error
}
