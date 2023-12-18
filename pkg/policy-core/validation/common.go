package validation

import (
	"context"
	"fmt"

	"github.com/weaveworks/policy-agent/pkg/policy-core/domain"
)

func matchEntity(entity domain.Entity, policy domain.Policy) bool {
	var matchKind bool
	var matchNamespace bool
	var matchLabel bool

	if len(policy.Targets.Kinds) == 0 {
		matchKind = true
	} else {
		resourceKind := entity.Kind
		for _, kind := range policy.Targets.Kinds {
			if resourceKind == kind {
				matchKind = true
				break
			}
		}
	}

	if len(policy.Targets.Namespaces) == 0 {
		matchNamespace = true
	} else {
		resourceNamespace := entity.Namespace
		for _, namespace := range policy.Targets.Namespaces {
			if resourceNamespace == namespace {
				matchNamespace = true
				break
			}
		}
	}

	if len(policy.Targets.Labels) == 0 {
		matchLabel = true
	} else {
	outer:
		for _, obj := range policy.Targets.Labels {
			for key, val := range obj {
				entityVal, ok := entity.Labels[key]
				if ok {
					if val != "*" && val != entityVal {
						continue
					}
					matchLabel = true
					break outer
				}
			}
		}
	}

	return matchKind && matchNamespace && matchLabel
}

// isExcluded evaluates the policy exclusion against the requested entity
func isExcluded(entity domain.Entity, policy domain.Policy) bool {
	var excludedNamespace bool
	var excludedResource bool
	var excludedLabel bool

	if len(policy.Exclude.Namespaces) == 0 {
		excludedNamespace = false
	} else {
		resourceNamespace := entity.Namespace
		for _, namespace := range policy.Exclude.Namespaces {
			if resourceNamespace == namespace {
				excludedNamespace = true
				break
			}
		}
	}

	if len(policy.Exclude.Resources) == 0 {
		excludedResource = false
	} else {
		resourceName := fmt.Sprintf("%s/%s", entity.Namespace, entity.Name)
		for _, resource := range policy.Exclude.Resources {
			if resourceName == resource {
				excludedResource = true
				break
			}
		}
	}

	if len(policy.Exclude.Labels) == 0 {
		excludedLabel = false
	} else {
	outer:
		for _, obj := range policy.Exclude.Labels {
			for key, val := range obj {
				entityVal, ok := entity.Labels[key]
				if ok {
					if val != "*" && val != entityVal {
						continue
					}
					excludedLabel = true
					break outer
				}
			}
		}
	}

	return excludedNamespace || excludedResource || excludedLabel
}

func writeToSinks(
	ctx context.Context,
	resultsSinks []domain.PolicyValidationSink,
	PolicyValidationSummary domain.PolicyValidationSummary,
	writeCompliance bool) {
	for _, resutsSink := range resultsSinks {
		if len(PolicyValidationSummary.Violations) > 0 {
			resutsSink.Write(ctx, PolicyValidationSummary.Violations)
		}
		if writeCompliance && len(PolicyValidationSummary.Compliances) > 0 {
			resutsSink.Write(ctx, PolicyValidationSummary.Compliances)
		}
	}
}
