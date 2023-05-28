mock:
	mockgen -package mock -destination domain/mock/policies.go github.com/weaveworks/weave-policy-agent/pkg/policy-core/domain PolicyValidationSink
	mockgen -package mock -destination domain/mock/sink.go github.com/weaveworks/weave-policy-agent/pkg/policy-core/domain PoliciesSource
	mockgen -package mock -destination validation/mock/mock.go github.com/weaveworks/weave-policy-agent/pkg/policy-core/validation Validator
