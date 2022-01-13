package logging

import (
	"context"

	"github.com/MagalixCorp/new-magalix-agent/validation"
	"github.com/MagalixTechnologies/core/logger"
)

type LogSink struct{}

func NewLogSink() *LogSink {
	return &LogSink{}
}

func (l *LogSink) Write(ctx context.Context, violations []validation.ValidationResult) error {
	for i := range violations {
		violation := violations[i]
		logger.Infow(
			"Got review result",
			"policy-id", violation.PolicyID,
			"entity-kind", violation.EntityKind,
			"entity-name", violation.EntityName,
			"status", violation.Status,
		)
	}
	return nil
}
