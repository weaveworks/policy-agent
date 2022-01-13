package validation

import (
	"context"

	"github.com/MagalixCorp/new-magalix-agent/pkg/domain"
)

type Validator interface {
	Validate(ctx context.Context, entity domain.Entity, source string) (*domain.ValidationSummary, error)
}
