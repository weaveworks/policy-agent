package logging

import (
	"context"

	"github.com/MagalixCorp/new-magalix-agent/pkg/domain"
	"github.com/MagalixTechnologies/core/logger"
)

type LogSink struct{}

func NewLogSink() *LogSink {
	return &LogSink{}
}

func (l *LogSink) Write(ctx context.Context, violations []domain.ValidationResult) error {
	for i := range violations {
		violation := violations[i]
		logger.Infow(
			"Got review result",
			"policy-id", violation.Policy.ID,
			"entity-kind", violation.Entity.Kind,
			"entity-name", violation.Entity.Name,
			"status", violation.Status,
		)
	}
	return nil
}
