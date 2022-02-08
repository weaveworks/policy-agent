package validation

import (
	"context"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
)

// Validator is responsible for validating policies
type Validator interface {
	// Validate returns validation results for the specified entity
	Validate(ctx context.Context, entity domain.Entity, source string) (*domain.ValidationSummary, error)
}
