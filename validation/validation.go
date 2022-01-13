package validation

import (
	"context"
	"errors"
	"fmt"
	"sync"

	opa "github.com/MagalixTechnologies/opa-core"
	uuid "github.com/MagalixTechnologies/uuid-go"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const PolicyQuery = "violation"

type Validator struct {
	policiesSource  PoliciesSource
	resultsSinks    []ValidationResultSink
	writeCompliance bool
	source          string
}

func NewValidator(
	policiesSource PoliciesSource,
	writeCompliance bool,
	source string,
	resultsSinks ...ValidationResultSink,
) *Validator {
	return &Validator{
		policiesSource:  policiesSource,
		resultsSinks:    resultsSinks,
		writeCompliance: writeCompliance,
		source:          source,
	}
}

func matchEntity(resource unstructured.Unstructured, policy Policy) bool {
	var matchKind bool
	var matchNamespace bool
	var matchLabel bool

	if len(policy.Targets.Kind) == 0 {
		matchKind = true
	} else {
		resourceKind := resource.GetKind()
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
		resourceNamespace := resource.GetNamespace()
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
				entityVal, ok := resource.GetLabels()[key]
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

func getPolicyParamsValues(parameters []PolicyParameters) map[string]interface{} {
	res := make(map[string]interface{})
	for _, param := range parameters {
		res[param.Name] = param.Default
	}
	return res
}

func (v *Validator) Validate(ctx context.Context, entity map[string]interface{}) (*ValidationSummary, error) {
	kubeEntity := unstructured.Unstructured{Object: entity}
	violations := make([]ValidationResult, 0)
	compliances := make([]ValidationResult, 0)
	var err error
	entityKind := kubeEntity.GetKind()
	entityName := kubeEntity.GetName()

	policies, err := v.policiesSource.GetPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("Failed to get policies from source, %w", err)
	}

	var enqueueGroup sync.WaitGroup
	var dequeueGroup sync.WaitGroup
	violationsChan := make(chan ValidationResult)
	compliancesChan := make(chan ValidationResult)
	errsChan := make(chan error)

	for i := range policies {
		enqueueGroup.Add(1)
		go (func(index int) {
			defer enqueueGroup.Done()
			policy := policies[index]
			match := matchEntity(kubeEntity, policy)
			if !match {
				return
			}
			opaPolicy, err := opa.Parse(policy.Code, PolicyQuery)
			if err != nil {
				errsChan <- fmt.Errorf("Failed to parse policy %s, %w", policy.ID, err)
			}
			var opaErr opa.OPAError
			res := ValidationResult{
				ID:         uuid.NewV4().String(),
				PolicyID:   policy.ID,
				EntityName: entityName,
				EntityKind: entityKind,
				Source:     v.source,
			}

			parameters := getPolicyParamsValues(policy.Parameters)
			err = opaPolicy.EvalGateKeeperCompliant(entity, parameters, PolicyQuery)
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

					msg := fmt.Sprintf("%s in %s %s", title, entityKind, entityName)
					res.Status = ValidationResultStatusViolating
					res.Message = msg
					violationsChan <- res
				} else {
					errsChan <- fmt.Errorf("unable to evaluate resource against policy. policy id: %s. %w", policy.ID, err)
				}

			} else {
				res.Status = ValidationResultStatusCompliant
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

	validationSummary := ValidationSummary{
		Violations:  violations,
		Compliances: compliances,
	}

	v.writeToSinks(ctx, validationSummary)

	return &validationSummary, nil
}

func (v *Validator) writeToSinks(ctx context.Context, validationSummary ValidationSummary) {
	for _, resutsSink := range v.resultsSinks {
		if len(validationSummary.Violations) > 0 {
			resutsSink.Write(ctx, validationSummary.Violations)
		}
		if v.writeCompliance && len(validationSummary.Compliances) > 0 {
			resutsSink.Write(ctx, validationSummary.Compliances)
		}
	}
}
