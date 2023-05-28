package validation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/weaveworks/weave-policy-agent/pkg/logger"
	opa "github.com/weaveworks/weave-policy-agent/pkg/opa-core"
	"github.com/weaveworks/weave-policy-agent/pkg/policy-core/domain"
	"github.com/weaveworks/weave-policy-agent/pkg/uuid-go"
)

const (
	PolicyQuery = "violation"
	maxWorkers  = 25
)

type OpaValidator struct {
	policiesSource  domain.PoliciesSource
	resultsSinks    []domain.PolicyValidationSink
	writeCompliance bool
	validationType  string
	accountID       string
	clusterID       string
	mutate          bool
}

// NewOPAValidator returns an opa validator to validate entities
func NewOPAValidator(
	policiesSource domain.PoliciesSource,
	writeCompliance bool,
	validationType string,
	accountID string,
	clusterID string,
	mutate bool,
	resultsSinks ...domain.PolicyValidationSink,
) *OpaValidator {
	return &OpaValidator{
		policiesSource:  policiesSource,
		resultsSinks:    resultsSinks,
		writeCompliance: writeCompliance,
		validationType:  validationType,
		accountID:       accountID,
		clusterID:       clusterID,
		mutate:          mutate,
	}
}

// Validate validate policies using opa library, implements validation.Validator
func (v *OpaValidator) Validate(ctx context.Context, entity domain.Entity, trigger string) (*domain.PolicyValidationSummary, error) {
	policies, err := v.policiesSource.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get policies from source: %w", err)
	}

	config, err := v.policiesSource.GetPolicyConfig(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("Failed to get policy config from source: %w", err)
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
			if !matchEntity(entity, policy) {
				return
			}

			opaPolicy, err := opa.Parse(policy.Code, PolicyQuery)
			if err != nil {
				errsChan <- fmt.Errorf("failed to parse policy %s: %w", policy.ID, err)
				return
			}

			parameters := map[string]interface{}{}
			if config == nil {
				parameters = policy.GetParametersMap()
			} else {
				policyConfig, policyConfigExists := config.Config[policy.ID]
				for i, policyParam := range policy.Parameters {
					parameters[policyParam.Name] = policyParam.Value
					if policyConfigExists {
						if configParam, ok := policyConfig.Parameters[policyParam.Name]; ok {
							logger.Infow(
								"overriding parameter",
								"policy", policy.ID,
								"parameter", policyParam.Name,
								"oldValue", policyParam.Value,
								"newValue", configParam.Value,
								"configRef", configParam.ConfigRef,
							)
							parameters[policyParam.Name] = configParam.Value
							policy.Parameters[i].Value = configParam.Value
							policy.Parameters[i].ConfigRef = configParam.ConfigRef
						}
					}
				}
			}

			var opaErr opa.OPAError
			err = opaPolicy.EvalGateKeeperCompliant(entity.Manifest, parameters, PolicyQuery)
			if err != nil {
				if errors.As(err, &opaErr) {
					dmsg := fmt.Sprintf(
						"%s in %s %s",
						policy.Name,
						strings.ToLower(entity.Kind),
						entity.Name,
					)

					details := opaErr.GetDetails()
					var occurrences []domain.Occurrence
					if arr, ok := details.([]interface{}); ok {
						for _, item := range arr {
							occurrences = append(occurrences, parseOccurrence(dmsg, item))
						}
					} else {
						occurrences = append(occurrences, parseOccurrence(dmsg, details))
					}

					message := fmt.Sprintf(
						"%s in %s %s (%d occurrences)",
						policy.Name,
						strings.ToLower(entity.Kind),
						entity.Name,
						len(occurrences),
					)
					result := domain.PolicyValidation{
						ID:          uuid.NewV4().String(),
						AccountID:   v.accountID,
						ClusterID:   v.clusterID,
						Policy:      policy,
						Entity:      entity,
						Type:        v.validationType,
						Trigger:     trigger,
						CreatedAt:   time.Now(),
						Message:     message,
						Status:      domain.PolicyValidationStatusViolating,
						Occurrences: occurrences,
					}
					violationsChan <- result

				} else {
					errsChan <- fmt.Errorf(
						"unable to evaluate resource against policy. policy id: %s. %w",
						policy.ID,
						err)
				}

			} else {
				result := domain.PolicyValidation{
					ID:        uuid.NewV4().String(),
					AccountID: v.accountID,
					ClusterID: v.clusterID,
					Policy:    policy,
					Entity:    entity,
					Type:      v.validationType,
					Trigger:   trigger,
					CreatedAt: time.Now(),
					Status:    domain.PolicyValidationStatusCompliant,
				}
				compliancesChan <- result
			}
		})(i)
	}
	violations := make([]domain.PolicyValidation, 0)
	dequeueGroup.Add(1)
	go func() {
		defer dequeueGroup.Done()
		for violation := range violationsChan {
			violations = append(violations, violation)
		}
	}()

	compliances := make([]domain.PolicyValidation, 0)
	dequeueGroup.Add(1)
	go func() {
		defer dequeueGroup.Done()
		for compliance := range compliancesChan {
			compliances = append(compliances, compliance)
		}
	}()

	var errs error
	dequeueGroup.Add(1)
	go func() {
		defer dequeueGroup.Done()
		for chanErr := range errsChan {
			errs = multierror.Append(errs, chanErr)
		}
	}()

	enqueueGroup.Wait()
	close(violationsChan)
	close(compliancesChan)
	close(errsChan)
	dequeueGroup.Wait()

	if errs != nil {
		return nil, fmt.Errorf(
			"encountered errors while validating policies against resource %s/%s: %w",
			entity.Kind,
			entity.Name,
			errs)
	}

	var mutationResult *domain.MutationResult
	var unmutatedViolations []domain.PolicyValidation

	if v.mutate {
		mutationResult, err = domain.NewMutationResult(entity)
		if err != nil {
			return nil, err
		}
		for i, violation := range violations {
			if !violation.Policy.Mutate {
				continue
			}
			occurrences, err := mutationResult.Mutate(violation.Occurrences)
			if err != nil {
				return nil, err
			}
			var unmutatedOccurrences []domain.Occurrence
			for _, occurrence := range occurrences {
				if !occurrence.Mutated {
					unmutatedOccurrences = append(unmutatedOccurrences, occurrence)
				}
			}
			if len(unmutatedOccurrences) == 0 {
				continue
			}
			violations[i].Occurrences = unmutatedOccurrences
			unmutatedViolations = append(unmutatedViolations, violation)
		}
	} else {
		unmutatedViolations = violations
	}

	PolicyValidationSummary := domain.PolicyValidationSummary{
		Violations:  unmutatedViolations,
		Compliances: compliances,
		Mutation:    mutationResult,
	}

	writeToSinks(ctx, v.resultsSinks, PolicyValidationSummary, v.writeCompliance)

	return &PolicyValidationSummary, nil
}

func parseOccurrence(msg string, in interface{}) domain.Occurrence {
	occurrence := domain.Occurrence{Message: msg}
	if v, ok := in.(map[string]interface{}); ok {
		if msg, ok := v["msg"].(string); ok {
			occurrence.Message = msg
		}
		if key, ok := v["violating_key"].(string); ok {
			occurrence.ViolatingKey = &key
		}
		occurrence.RecommendedValue = v["recommended_value"]
	}
	return occurrence
}
