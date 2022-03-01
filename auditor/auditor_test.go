package auditor

import (
	"context"
	"testing"
	"time"

	entitiesmock "github.com/MagalixCorp/magalix-policy-agent/entities/mock"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	validationmock "github.com/MagalixCorp/magalix-policy-agent/pkg/validation/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

const (
	auditInterval = 2 * time.Second
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
			got := NewAuditController(validator, auditInterval, entitiesSource)
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
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
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
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
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
			a := NewAuditController(validator, auditInterval, entitiesSource)
			auditEvent := AuditEvent{Type: test.args.auditType}
			a.doAudit(context.Background(), auditEvent)
		})
	}
}

func assertEvent(c chan AuditEvent, assert *require.Assertions, target AuditEventType) {
	var event *AuditEvent
	for event == nil {
		select {
		case e := <-c:
			event = &e
		}
	}
	assert.Equal(target, event.Type)
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

			auditEventChan := make(chan AuditEvent, 1)
			a := NewAuditController(validator, auditInterval, entitiesSource)
			a.RegisterAuditEventListener(func(ctx context.Context, auditEvent AuditEvent) {
				auditEventChan <- auditEvent
			})
			ctx, cancelFunc := context.WithCancel(context.Background())
			go a.Run(ctx)
			a.Audit(test.args.auditType, test.args.data)
			assertEvent(auditEventChan, assert, test.args.auditType)

			assertEvent(auditEventChan, assert, AuditEventTypePeriodical)

			cancelFunc()
		})
	}
}
