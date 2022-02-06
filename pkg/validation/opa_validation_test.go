package validation

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/validation/testdata"
	mockpolicy "github.com/MagalixCorp/magalix-policy-agent/policies/mock"
	mocksink "github.com/MagalixCorp/magalix-policy-agent/sink/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestNewOpaValidator(t *testing.T) {
	type args struct {
		policiesSource  domain.PoliciesSource
		writeCompliance bool
		resultsSinks    []domain.ValidationResultSink
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	policiesSource := mockpolicy.NewMockPoliciesSource(ctrl)
	sink := mocksink.NewMockValidationResultSink(ctrl)
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
				resultsSinks:    []domain.ValidationResultSink{sink},
			},
			want: &OpaValidator{
				policiesSource:  policiesSource,
				writeCompliance: true,
				resultsSinks:    []domain.ValidationResultSink{sink},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewOpaValidator(tt.args.policiesSource, tt.args.writeCompliance, tt.args.resultsSinks...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOpaValidator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func cmpValidationResult(arg1, arg2 domain.ValidationResult) bool {
	if arg1.Entity.ID != arg2.Entity.ID {
		return false
	}

	return arg1.Source == arg2.Source && arg1.Status == arg2.Status
}

func TestOpaValidator_Validate(t *testing.T) {
	type init struct {
		loadStubs       func(*mockpolicy.MockPoliciesSource, *mocksink.MockValidationResultSink)
		writeCompliance bool
	}

	entityText := testdata.Entity
	source := "unit-test"
	entity, _ := domain.NewEntityFromStringSpec(entityText)
	compliantEntity, _ := domain.NewEntityFromStringSpec(testdata.CompliantEntity)
	tests := []struct {
		name    string
		init    init
		entity  domain.Entity
		want    *domain.ValidationSummary
		wantErr bool
	}{
		{
			name: "default test",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mockpolicy.MockPoliciesSource, sink *mocksink.MockValidationResultSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["imageTag"],
						testdata.Policies["missingOwner"],
					}, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.ValidationSummary{
				Violations: []domain.ValidationResult{
					{
						Policy: testdata.Policies["imageTag"],
						Entity: entity,
						Source: source,
						Status: domain.ValidationResultStatusViolating,
					},
					{
						Policy: testdata.Policies["missingOwner"],
						Entity: entity,
						Source: source,
						Status: domain.ValidationResultStatusViolating,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error getting policies",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mockpolicy.MockPoliciesSource, sink *mocksink.MockValidationResultSink) {
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
				loadStubs: func(policiesSource *mockpolicy.MockPoliciesSource, sink *mocksink.MockValidationResultSink) {
					missingOwner := testdata.Policies["missingOwner"]
					missingOwner.Targets = domain.PolicyTargets{Kind: []string{"Deployment"}}
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{Kind: []string{"ReplicaSet"}}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						missingOwner,
						imageTag,
					}, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.ValidationSummary{
				Violations: []domain.ValidationResult{
					{
						Policy: testdata.Policies["missingOwner"],
						Entity: entity,
						Source: source,
						Status: domain.ValidationResultStatusViolating,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "entity namespace matching",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mockpolicy.MockPoliciesSource, sink *mocksink.MockValidationResultSink) {
					missingOwner := testdata.Policies["missingOwner"]
					missingOwner.Targets = domain.PolicyTargets{Namespace: []string{"unit-testing"}}
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{Namespace: []string{"bad-namespace"}}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						missingOwner,
						imageTag,
					}, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.ValidationSummary{
				Violations: []domain.ValidationResult{
					{
						Policy: testdata.Policies["missingOwner"],
						Entity: entity,
						Source: source,
						Status: domain.ValidationResultStatusViolating,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "entity labels matching",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mockpolicy.MockPoliciesSource, sink *mocksink.MockValidationResultSink) {
					missingOwner := testdata.Policies["missingOwner"]
					missingOwner.Targets = domain.PolicyTargets{Label: []map[string]string{{"app": "nginx"}}}
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{Label: []map[string]string{{"app": "notfound"}}}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						missingOwner,
						imageTag,
					}, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.ValidationSummary{
				Violations: []domain.ValidationResult{
					{
						Policy: testdata.Policies["missingOwner"],
						Entity: entity,
						Source: source,
						Status: domain.ValidationResultStatusViolating,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "matching multiple",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mockpolicy.MockPoliciesSource, sink *mocksink.MockValidationResultSink) {
					missingOwner := testdata.Policies["missingOwner"]
					missingOwner.Targets = domain.PolicyTargets{
						Label:     []map[string]string{{"app": "nginx"}},
						Namespace: []string{"unit-testing"},
						Kind:      []string{"Deployment"},
					}
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{
						Label:     []map[string]string{{"app": "nginx"}},
						Namespace: []string{"bad-namespace"},
						Kind:      []string{"Deployment"},
					}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						missingOwner,
						imageTag,
					}, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: entity,
			want: &domain.ValidationSummary{
				Violations: []domain.ValidationResult{
					{
						Policy: testdata.Policies["missingOwner"],
						Entity: entity,
						Source: source,
						Status: domain.ValidationResultStatusViolating,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no mathching no writes to sink",
			init: init{
				writeCompliance: true,
				loadStubs: func(policiesSource *mockpolicy.MockPoliciesSource, sink *mocksink.MockValidationResultSink) {
					imageTag := testdata.Policies["imageTag"]
					imageTag.Targets = domain.PolicyTargets{Label: []map[string]string{{"app": "notfound"}}}
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						imageTag,
					}, nil)
					// expect 0 calls to sink write, no compliance or violation
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(0).Return(nil)
				},
			},
			entity:  entity,
			want:    &domain.ValidationSummary{},
			wantErr: false,
		},
		{
			name: "compliant entity",
			init: init{
				writeCompliance: true,
				loadStubs: func(policiesSource *mockpolicy.MockPoliciesSource, sink *mocksink.MockValidationResultSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["imageTag"],
						testdata.Policies["missingOwner"],
					}, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(1).Return(nil)
				},
			},
			entity: compliantEntity,
			want: &domain.ValidationSummary{
				Compliances: []domain.ValidationResult{
					{
						Policy: testdata.Policies["imageTag"],
						Entity: compliantEntity,
						Source: source,
						Status: domain.ValidationResultStatusCompliant,
					},
					{
						Policy: testdata.Policies["missingOwner"],
						Entity: compliantEntity,
						Source: source,
						Status: domain.ValidationResultStatusCompliant,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "error loading policy code",
			init: init{
				writeCompliance: false,
				loadStubs: func(policiesSource *mockpolicy.MockPoliciesSource, sink *mocksink.MockValidationResultSink) {
					policiesSource.EXPECT().GetAll(gomock.Any()).
						Times(1).Return([]domain.Policy{
						testdata.Policies["badPolicyCode"],
					}, nil)
					sink.EXPECT().Write(gomock.Any(), gomock.Any()).
						Times(0).Return(nil)
				},
			},
			entity:  entity,
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := require.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			policiesSource := mockpolicy.NewMockPoliciesSource(ctrl)
			sink := mocksink.NewMockValidationResultSink(ctrl)
			tt.init.loadStubs(policiesSource, sink)
			v := &OpaValidator{
				policiesSource:  policiesSource,
				resultsSinks:    []domain.ValidationResultSink{sink},
				writeCompliance: tt.init.writeCompliance,
			}
			got, err := v.Validate(context.Background(), tt.entity, source)
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
							cmpValidationResult(gotViolation, wantViolation),
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
							cmpValidationResult(gotCompliance, wantCompliance),
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
