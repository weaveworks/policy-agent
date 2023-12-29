package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/policy-agent/pkg/policy-core/domain"
	"github.com/weaveworks/policy-agent/pkg/policy-core/domain/mock"
	"github.com/weaveworks/policy-agent/pkg/policy-core/validation/testdata"
)

func TestNewOPAValidator(t *testing.T) {
	type args struct {
		policiesSource  domain.PoliciesSource
		writeCompliance bool
		resultsSinks    []domain.PolicyValidationSink
		validationType  string
		accountID       string
		clusterID       string
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	policiesSource := mock.NewMockPoliciesSource(ctrl)
	sink := mock.NewMockPolicyValidationSink(ctrl)
	tests := []struct {
		name string
		args args
		want *OpaValidator
	}{
		{
			name: "default test",
			args: args{
				policiesSource:  policiesSource,
				writeCompliance: true,
				resultsSinks:    []domain.PolicyValidationSink{sink},
				validationType:  "TestValidate",
				accountID:       "account-id",
				clusterID:       "cluster-id",
			},
			want: &OpaValidator{
				policiesSource:  policiesSource,
				writeCompliance: true,
				resultsSinks:    []domain.PolicyValidationSink{sink},
				validationType:  "TestValidate",
				accountID:       "account-id",
				clusterID:       "cluster-id",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewOPAValidator(tt.args.policiesSource, tt.args.writeCompliance, tt.args.validationType, tt.args.accountID, tt.args.clusterID, false, tt.args.resultsSinks...); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("NewOPAValidator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func cmpPolicyValidation(arg1, arg2 domain.PolicyValidation) bool {
	if arg1.Entity.ID != arg2.Entity.ID {
		return false
	}

	if arg1.Message != arg2.Message {
		return false
	}

	if len(arg1.Occurrences) == len(arg2.Occurrences) {
		for i, occurrence := range arg1.Occurrences {
			if occurrence.Message != arg2.Occurrences[i].Message {
				return false
			}
		}
	} else {
		return false
	}

	if arg1.Enforced != arg2.Enforced {
		return false
	}

	return arg1.Type == arg2.Type &&
		arg1.Trigger == arg2.Trigger &&
		arg1.Status == arg2.Status
}

func getEntityFromStringSpec(entityStringSpec string) (domain.Entity, error) {
	var entitySpec map[string]interface{}
	err := json.Unmarshal([]byte(entityStringSpec), &entitySpec)
	if err != nil {
		return domain.Entity{}, fmt.Errorf("invalid string format: %w", err)
	}
	return domain.NewEntityFromSpec(entitySpec), nil
}

func TestOpaValidator_Mutate(t *testing.T) {
	type init struct {
		loadStubs       func(*mock.MockPoliciesSource, *mock.MockPolicyValidationSink)
		writeCompliance bool
	}
	assert := require.New(t)
	entityText := testdata.Entity
	validationType := "unit-test"
	entity, err := getEntityFromStringSpec(entityText)
	assert.Nil(err)

	tests := []struct {
		name        string
		init        init
		entity      domain.Entity
		violations  int
		occurrences int
	}{
		{
			name: "mutate all violations",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["missingOwner"],
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(0).Return(nil)
				},
			},
			entity:      entity,
			occurrences: 0,
			violations:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			policiesSource := mock.NewMockPoliciesSource(ctrl)
			sink := mock.NewMockPolicyValidationSink(ctrl)
			tt.init.loadStubs(policiesSource, sink)
			v := &OpaValidator{
				policiesSource:  policiesSource,
				resultsSinks:    []domain.PolicyValidationSink{sink},
				writeCompliance: tt.init.writeCompliance,
				validationType:  validationType,
				mutate:          true,
			}
			result, err := v.Validate(context.Background(), tt.entity, validationType)
			assert.Nil(err)

			b, _ := result.Mutation.NewResource()
			fmt.Println(string(b))

			assert.NotNil(result.Mutation)
			assert.Equal(tt.violations, len(result.Violations))
			if tt.occurrences > 0 {
				assert.Equal(tt.occurrences, len(result.Violations[0].Occurrences))
			}
		})
	}
}

func TestOpaValidator_Validate(t *testing.T) {
	type init struct {
		loadStubs       func(*mock.MockPoliciesSource, *mock.MockPolicyValidationSink)
		writeCompliance bool
	}
	assert := require.New(t)

	entityText := testdata.Entity
	validationType := "unit-test"
	entity, err := getEntityFromStringSpec(entityText)
	assert.Nil(err)
	compliantEntity, err := getEntityFromStringSpec(testdata.CompliantEntity)
	assert.Nil(err)
	tests := []struct {
		name    string
		init    init
		entity  domain.Entity
		want    *domain.PolicyValidationSummary
		wantErr bool
	}{
		{
			name: "default test",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["imageTag"],
						testdata.Policies["missingOwner"],
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{
					{
						Policy:  testdata.Policies["imageTag"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Using latest image tag in container in deployment nginx-deployment (1 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "Image contains unapproved tag 'latest'",
							},
						},
						Enforced: false,
					},
					{
						Policy:  testdata.Policies["missingOwner"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Missing owner label in metadata in deployment nginx-deployment (1 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "you are missing a label with the key 'owner'",
							},
						},
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error getting policies",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return(nil, fmt.Errorf(""))
					// expect 0 calls to sink write
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(0).Return(nil)
				},
			},
			entity:  entity,
			want:    nil,
			wantErr: true,
		},
		{
			name: "entity kind matching",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					missingOwner := testdata.Policies["missingOwner"]
					missingOwner.Targets = domain.PolicyTargets{Kinds: []string{"Deployment"}}
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{Kinds: []string{"ReplicaSet"}}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						missingOwner,
						imageTag,
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{
					{
						Policy:  testdata.Policies["missingOwner"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Missing owner label in metadata in deployment nginx-deployment (1 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "you are missing a label with the key 'owner'",
							},
						},
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "entity namespace matching",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					missingOwner := testdata.Policies["missingOwner"]
					missingOwner.Targets = domain.PolicyTargets{Namespaces: []string{"unit-testing"}}
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{Namespaces: []string{"bad-namespace"}}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						missingOwner,
						imageTag,
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{
					{
						Policy:  testdata.Policies["missingOwner"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Missing owner label in metadata in deployment nginx-deployment (1 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "you are missing a label with the key 'owner'",
							},
						},
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "entity labels matching",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					missingOwner := testdata.Policies["missingOwner"]
					missingOwner.Targets = domain.PolicyTargets{Labels: []map[string]string{{"app": "nginx"}}}
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{Labels: []map[string]string{{"app": "notfound"}}}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						missingOwner,
						imageTag,
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{
					{
						Policy:  testdata.Policies["missingOwner"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Missing owner label in metadata in deployment nginx-deployment (1 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "you are missing a label with the key 'owner'",
							},
						},
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple policies only one matching",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					missingOwner := testdata.Policies["missingOwner"]
					missingOwner.Targets = domain.PolicyTargets{
						Labels:     []map[string]string{{"app": "nginx"}},
						Namespaces: []string{"unit-testing"},
						Kinds:      []string{"Deployment"},
					}
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{
						Labels:     []map[string]string{{"app": "nginx"}},
						Namespaces: []string{"bad-namespace"},
						Kinds:      []string{"Deployment"},
					}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						missingOwner,
						imageTag,
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{
					{
						Policy:  testdata.Policies["missingOwner"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Missing owner label in metadata in deployment nginx-deployment (1 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "you are missing a label with the key 'owner'",
							},
						},
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no mathching no writes to sink",
			init: init{
				writeCompliance: true,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{Labels: []map[string]string{{"app": "notfound"}}}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						imageTag,
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					// expect 0 calls to sink write, no compliance or violation
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(0).Return(nil)
				},
			},
			entity:  entity,
			want:    &domain.PolicyValidationSummary{},
			wantErr: false,
		},
		{
			name: "compliant entity",
			init: init{
				writeCompliance: true,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["imageTag"],
						testdata.Policies["missingOwner"],
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: compliantEntity,
			want: &domain.PolicyValidationSummary{
				Compliances: []domain.PolicyValidation{
					{
						Policy:   testdata.Policies["imageTag"],
						Entity:   compliantEntity,
						Type:     validationType,
						Status:   domain.PolicyValidationStatusCompliant,
						Trigger:  validationType,
						Enforced: false,
					},
					{
						Policy:   testdata.Policies["missingOwner"],
						Entity:   compliantEntity,
						Type:     validationType,
						Status:   domain.PolicyValidationStatusCompliant,
						Trigger:  validationType,
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple violation",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["runningAsRoot"],
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{
					{
						Policy:  testdata.Policies["runningAsRoot"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Container Running As Root in deployment nginx-deployment (2 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "Container spec.template.spec.containers[0].securityContext.runAsNonRoot should be set to true",
							},
							{
								Message: "Pod spec.template.spec.securityContext.runAsNonRoot should be set to true",
							},
						},
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "pass because of policy config value",
			init: init{
				writeCompliance: true,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["replicaCount"],
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(&domain.PolicyConfig{
						Config: map[string]domain.PolicyConfigConfig{
							testdata.Policies["replicaCount"].ID: {
								Parameters: map[string]domain.PolicyConfigParameter{
									"replica_count": {
										Value:     3,
										ConfigRef: "my-config",
									},
								},
							},
						},
					}, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Compliances: []domain.PolicyValidation{
					{
						Policy:   testdata.Policies["replicaCount"],
						Entity:   compliantEntity,
						Type:     validationType,
						Status:   domain.PolicyValidationStatusCompliant,
						Trigger:  validationType,
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "fail because of policy config value",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["replicaCount"],
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(&domain.PolicyConfig{
						Config: map[string]domain.PolicyConfigConfig{
							testdata.Policies["replicaCount"].ID: {
								Parameters: map[string]domain.PolicyConfigParameter{
									"replica_count": {
										Value:     5,
										ConfigRef: "my-config",
									},
								},
							},
						},
					}, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{
					{
						Policy:  testdata.Policies["replicaCount"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Minimum replica count in deployment nginx-deployment (1 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "Replica count must be greater than or equal to '5'; found '3'.",
							},
						},
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error loading policy code",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["badPolicyCode"],
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(0).Return(nil)
				},
			},
			entity:  entity,
			want:    nil,
			wantErr: true,
		},
		{
			name: "default test with enforce is true",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["imageTagEnforced"],
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{
					{
						Policy:  testdata.Policies["imageTagEnforced"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Using latest image tag in container in deployment nginx-deployment (1 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "Image contains unapproved tag 'latest'",
							},
						},
						Enforced: true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "entity namespace exclusion",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					imageTag := testdata.Policies["imageTagExcludedNamespaces"]
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						imageTag,
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{},
			},
			wantErr: false,
		},
		{
			name: "entity labels exclusion",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					imageTag := testdata.Policies["imageTagExcludedLabels"]
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						imageTag,
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{},
			},
			wantErr: false,
		},
		{
			name: "entity resource exclusion",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mock.MockPoliciesSource, sink *mock.MockPolicyValidationSink) {
					missingOwner := testdata.Policies["missingOwner"]
					imageTag := testdata.Policies["imageTagExcludedResources"]
					imageTag.Exclude.Resources = []string{"unit-testing/nginx-deployment"}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						missingOwner,
						imageTag,
					}, nil)
					policiesSource.EXPECT().GetPolicyConfig(gomock.Any(), gomock.Any()).
						Times(1).Return(nil, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.PolicyValidationSummary{
				Violations: []domain.PolicyValidation{
					{
						Policy:  testdata.Policies["missingOwner"],
						Entity:  entity,
						Type:    validationType,
						Status:  domain.PolicyValidationStatusViolating,
						Trigger: validationType,
						Message: "Missing owner label in metadata in deployment nginx-deployment (1 occurrences)",
						Occurrences: []domain.Occurrence{
							{
								Message: "you are missing a label with the key 'owner'",
							},
						},
						Enforced: false,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			policiesSource := mock.NewMockPoliciesSource(ctrl)
			sink := mock.NewMockPolicyValidationSink(ctrl)
			tt.init.loadStubs(policiesSource, sink)
			v := &OpaValidator{
				policiesSource:  policiesSource,
				resultsSinks:    []domain.PolicyValidationSink{sink},
				writeCompliance: tt.init.writeCompliance,
				validationType:  validationType,
			}
			got, err := v.Validate(context.Background(), tt.entity, validationType)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if err != nil {
				assert.Fail("validator test failed", "got unexcpected error, %s", err)
			}
			assert.Equal(len(got.Violations), len(tt.want.Violations))
			assert.Equal(len(got.Compliances), len(tt.want.Compliances))

			for _, wantViolation := range tt.want.Violations {
				found := false
				for _, gotViolation := range got.Violations {
					if gotViolation.Policy.ID == wantViolation.Policy.ID {
						found = true
						assert.True(
							cmpPolicyValidation(gotViolation, wantViolation),
							"gotten violation not as expected for policy %s",
							wantViolation.Policy.ID,
						)
					}
				}
				assert.True(found, "did not find violation for policy %s", wantViolation.Policy.Name)
			}

			for _, wantCompliance := range tt.want.Compliances {
				found := false
				for _, gotCompliance := range got.Compliances {
					if gotCompliance.Policy.ID == wantCompliance.Policy.ID {
						found = true
						assert.True(
							cmpPolicyValidation(gotCompliance, wantCompliance),
							"gotten compliance not as expected for policy %s",
							wantCompliance.Policy.ID,
						)
					}
				}
				assert.True(found, "did not find compliance for policy %s", wantCompliance.Policy.Name)
			}
		})
	}
}
