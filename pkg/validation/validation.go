package validation

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	opa "github.com/MagalixTechnologies/opa-core"
	uuid "github.com/MagalixTechnologies/uuid-go"
)

const PolicyQuery = "violation"

type OpaValidator struct {
	policiesSource  domain.PoliciesSource
	resultsSinks    []domain.ValidationResultSink
	writeCompliance bool
}

func NewOpaValidator(
	policiesSource domain.PoliciesSource,
	writeCompliance bool,
	resultsSinks ...domain.ValidationResultSink,
) *OpaValidator {
	return &OpaValidator{
		policiesSource:  policiesSource,
		resultsSinks:    resultsSinks,
		writeCompliance: writeCompliance,
	}
}

func matchEntity(entity domain.Entity, policy domain.Policy) bool {
	var matchKind bool
	var matchNamespace bool
	var matchLabel bool

	if len(policy.Targets.Kind) == 0 {
		matchKind = true
	} else {
		resourceKind := entity.Kind
		for _, kind := range policy.Targets.Kind {
			if resourceKind == kind {
				matchKind = true
				break
			}
		}
	}

	if len(policy.Targets.Namespace) == 0 {
		matchNamespace = true
	} else {
		resourceNamespace := entity.Namespace
		for _, namespace := range policy.Targets.Namespace {
			if resourceNamespace == namespace {
				matchNamespace = true
				break
			}
		}
	}

	if len(policy.Targets.Label) == 0 {
		matchLabel = true
	} else {
	outer:
		for _, obj := range policy.Targets.Label {
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

func (v *OpaValidator) Validate(ctx context.Context, entity domain.Entity, source string) (*domain.ValidationSummary, error) {
	violations := make([]domain.ValidationResult, 0)
	compliances := make([]domain.ValidationResult, 0)
	var err error

	policies, err := v.policiesSource.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get policies from source, %w", err)
	}

	var enqueueGroup sync.WaitGroup
	var dequeueGroup sync.WaitGroup
	violationsChan := make(chan domain.ValidationResult)
	compliancesChan := make(chan domain.ValidationResult)
	errsChan := make(chan error)

	for i := range policies {
		enqueueGroup.Add(1)
		go (func(index int) {
			defer enqueueGroup.Done()
			policy := policies[index]
			match := matchEntity(entity, policy)
			if !match {
				return
			}
			opaPolicy, err := opa.Parse(policy.Code, PolicyQuery)
			if err != nil {
				errsChan <- fmt.Errorf("Failed to parse policy %s, %w", policy.ID, err)
			}
			var opaErr opa.OPAError
			res := domain.ValidationResult{
				ID:     uuid.NewV4().String(),
				Policy: policy,
				Entity: entity,
				Source: source,
			}

			parameters := policy.GetParametersMap()
			err = opaPolicy.EvalGateKeeperCompliant(entity.Spec, parameters, PolicyQuery)
			if err != nil {
				if errors.As(err, &opaErr) {
					details := make(map[string]interface{})
					detailsInt := opaErr.GetDetails()
					detailsMap, ok := detailsInt.(map[string]interface{})
					if ok {
						details = detailsMap
					}

					var title string
					if msg, ok := details["msg"]; ok {
						title = msg.(string)
					} else {
						title = policy.Name
					}

					msg := fmt.Sprintf("%s in %s %s. Policy: %s", title, entity.Kind, entity.Name, policy.ID)
					res.Status = domain.ValidationResultStatusViolating
					res.Message = msg
					violationsChan <- res
				} else {
					errsChan <- fmt.Errorf("unable to evaluate resource against policy. policy id: %s. %w", policy.ID, err)
				}

			} else {
				res.Status = domain.ValidationResultStatusCompliant
				compliancesChan <- res

			}
		})(i)
	}
	dequeueGroup.Add(1)
	go func() {
		defer dequeueGroup.Done()
		for violation := range violationsChan {
			violations = append(violations, violation)
		}
	}()

	dequeueGroup.Add(1)
	go func() {
		defer dequeueGroup.Done()
		for compliance := range compliancesChan {
			compliances = append(compliances, compliance)
		}
	}()

	dequeueGroup.Add(1)
	go func() {
		defer dequeueGroup.Done()
		for chanErr := range errsChan {
			err = fmt.Errorf("%w;%s", err, chanErr)
		}
	}()

	enqueueGroup.Wait()
	close(violationsChan)
	close(compliancesChan)
	close(errsChan)
	dequeueGroup.Wait()

	if err != nil {
		return nil, fmt.Errorf("Encountered errors while validating policies, %w", err)
	}

	validationSummary := domain.ValidationSummary{
		Violations:  violations,
		Compliances: compliances,
	}

	v.writeToSinks(ctx, validationSummary)

	return &validationSummary, nil
}

func (v *OpaValidator) writeToSinks(ctx context.Context, validationSummary domain.ValidationSummary) {
	for _, resutsSink := range v.resultsSinks {
		if len(validationSummary.Violations) > 0 {
			resutsSink.Write(ctx, validationSummary.Violations)
		}
		if v.writeCompliance && len(validationSummary.Compliances) > 0 {
			resutsSink.Write(ctx, validationSummary.Compliances)
		}
	}
}
