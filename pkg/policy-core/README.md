# Policy Core

Contains policy validation tools and defines its expected domain objects and interfaces used by Weave Policy Agent.

## Policy Domain

Defines the structures for `Policy` objects (Policy, Entity, PolicySet, PolicyConfig, PolicyValidation) also the mocking tools for unit testing.

### Generate mock data

After making any changes to the policy domain or policy validator, use the following command to generate mock data

```bash
make mock
```

## Policy Validation

Validates specific entity against chosen policies also can push the validation result to different sinks. To use the validator call a new instance from `OPAValidator` with the following method then call the `validate` method

```go
// NewOPAValidator returns an opa validator to validate entities
validator := NewOPAValidator(
	policiesSource domain.PoliciesSource,
	writeCompliance bool,
	validationType string,
	accountID string,
	clusterID string,
	mutate bool,
	resultsSinks ...domain.PolicyValidationSink,
)
validator.validate(ctx context.Context, entity domain.Entity, trigger string)
```
