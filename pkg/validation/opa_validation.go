package validation

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	opa "github.com/MagalixTechnologies/opa-core"
	uuid "github.com/MagalixTechnologies/uuid-go"
)

const (
	PolicyQuery = "violation"
	maxWorkers  = 50
)

type OpaValidator struct {
	policiesSource  domain.PoliciesSource
	resultsSinks    []domain.PolicyValidationSink
	writeCompliance bool
}

// NewOpaValidator returns an opa validator to validate entities
func NewOpaValidator(
	policiesSource domain.PoliciesSource,
	writeCompliance bool,
	resultsSinks ...domain.PolicyValidationSink,
) *OpaValidator {
	return &OpaValidator{
		policiesSource:  policiesSource,
		resultsSinks:    resultsSinks,
		writeCompliance: writeCompliance,
	}
}

// Validate validate policies using opa library, implements validation.Validator
func (v *OpaValidator) Validate(ctx context.Context, entity domain.Entity, validationType, trigger string) (*domain.PolicyValidationSummary, error) {
	violations := make([]domain.PolicyValidation, 0)
	compliances := make([]domain.PolicyValidation, 0)
	var err error

	policies, err := v.policiesSource.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get policies from source, %w", err)
	}

	var enqueueGroup sync.WaitGroup
	var dequeueGroup sync.WaitGroup
	violationsChan := make(chan domain.PolicyValidation, len(policies))
	compliancesChan := make(chan domain.PolicyValidation, len(policies))
	errsChan := make(chan error, len(policies))
	bound := make(chan struct{}, maxWorkers)

	for i := range policies {
		bound <- struct{}{}
		enqueueGroup.Add(1)
		go (func(index int) {
			defer func() {
				<-bound
				enqueueGroup.Done()
			}()
			policy := policies[index]
			match := matchEntity(entity, policy)
			if !match {
				return
			}
			opaPolicy, err := opa.Parse(policy.Code, PolicyQuery)
			if err != nil {
				errsChan <- fmt.Errorf("Failed to parse policy %s, %w", policy.ID, err)
				return
			}
			var opaErr opa.OPAError
			res := domain.PolicyValidation{
				ID:        uuid.NewV4().String(),
				Policy:    policy,
				Entity:    entity,
				Type:      validationType,
				CreatedAt: time.Now(),
			}

			parameters := policy.GetParametersMap()
			err = opaPolicy.EvalGateKeeperCompliant(entity.Manifest, parameters, PolicyQuery)
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
					res.Status = domain.PolicyValidationStatusViolating
					res.Message = msg

					violationsChan <- res
				} else {
					errsChan <- fmt.Errorf("unable to evaluate resource against policy. policy id: %s. %w", policy.ID, err)
				}

			} else {
				res.Status = domain.PolicyValidationStatusCompliant
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

	PolicyValidationSummary := domain.PolicyValidationSummary{
		Violations:  violations,
		Compliances: compliances,
	}
	writeToSinks(ctx, v.resultsSinks, PolicyValidationSummary, v.writeCompliance)

	return &PolicyValidationSummary, nil
}
