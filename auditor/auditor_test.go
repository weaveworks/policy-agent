package auditor

import (
	"testing"

	entitiesmock "github.com/MagalixCorp/magalix-policy-agent/entities/mock"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	validationmock "github.com/MagalixCorp/magalix-policy-agent/pkg/validation/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestNewAuditController(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validator := validationmock.NewMockValidator(ctrl)
	entitiesSource := entitiesmock.NewMockEntitiesSource(ctrl)
	tests := []struct {
		name string
		want *AuditorController
	}{
		{
			name: "standard test",
			want: &AuditorController{
				entitiesSources: []domain.EntitiesSource{entitiesSource},
				validator:       validator,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := require.New(t)
			got := NewAuditController(validator, entitiesSource)
			assert.Equal(test.want.entitiesSources, got.entitiesSources, "unexpected auditor entities source")
			assert.Equal(test.want.validator, got.validator, "unexpected auditor validator")
		})
	}
}

func TestAuditorController_doAudit(t *testing.T) {
	type args struct {
		auditType AuditEventType
	}
	tests := []struct {
		name      string
		args      args
		loadStubs func(*validationmock.MockValidator, *entitiesmock.MockEntitiesSource)
	}{
		{
			name: "standard test",
			args: args{
				auditType: AuditEventTypeInitial,
			},
			loadStubs: func(val *validationmock.MockValidator, ent *entitiesmock.MockEntitiesSource) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).Return(&domain.PolicyValidationSummary{}, nil)
				ent.EXPECT().List(gomock.Any(), gomock.Any()).
					Times(1).Return(&domain.EntitiesList{
					HasNext: false,
					Data: []domain.Entity{
						{
							Name: "test",
							Kind: "Deployment",
						},
					},
				}, nil)
			},
		},
		{
			name: "list using pagination",
			args: args{
				auditType: AuditEventTypeInitial,
			},
			loadStubs: func(val *validationmock.MockValidator, ent *entitiesmock.MockEntitiesSource) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(2).Return(&domain.PolicyValidationSummary{}, nil)
				ent.EXPECT().List(gomock.Any(), gomock.Any()).
					Times(1).Return(&domain.EntitiesList{
					HasNext: true,
					Data: []domain.Entity{
						{
							Name: "test",
							Kind: "Deployment",
						},
					},
				}, nil)
				ent.EXPECT().List(gomock.Any(), gomock.Any()).
					Times(1).Return(&domain.EntitiesList{
					HasNext: false,
					Data: []domain.Entity{
						{
							Name: "test",
							Kind: "Deployment",
						},
					},
				}, nil)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			validator := validationmock.NewMockValidator(ctrl)
			entitiesSource := entitiesmock.NewMockEntitiesSource(ctrl)
			test.loadStubs(validator, entitiesSource)
			a := NewAuditController(validator, entitiesSource)
			a.doAudit(test.args.auditType)
		})
	}
}

func TestAuditorController_Audit(t *testing.T) {
	type args struct {
		auditType AuditEventType
		data      interface{}
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "standard test",
			args: args{
				auditType: AuditEventTypePeriodical,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := require.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			validator := validationmock.NewMockValidator(ctrl)
			entitiesSource := entitiesmock.NewMockEntitiesSource(ctrl)
			auditEvent := make(chan AuditEvent, 1)
			a := &AuditorController{
				validator:       validator,
				entitiesSources: []domain.EntitiesSource{entitiesSource},
				auditEvent:      auditEvent}
			a.Audit(test.args.auditType, test.args.data)
			var event AuditEvent
			select {
			case e := <-auditEvent:
				event = e
			default:
				assert.Fail("failed to validate audit event")
			}
			assert.Equal(test.args.auditType, event.Type)
		})
	}
}
