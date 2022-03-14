package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/MagalixCorp/magalix-policy-agent/internal/admission/testdata"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/validation"
	validationmock "github.com/MagalixCorp/magalix-policy-agent/pkg/validation/mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlAdmission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestNewAdmissionHandler(t *testing.T) {
	type args struct {
		logLevel  string
		validator validation.Validator
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	validator := validationmock.NewMockValidator(ctrl)
	tests := []struct {
		name string
		args args
		want *AdmissionHandler
	}{
		{
			name: "default test",
			args: args{
				logLevel:  "info",
				validator: validator,
			},
			want: &AdmissionHandler{
				logLevel:  "info",
				validator: validator,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAdmissionHandler(tt.args.logLevel, tt.args.validator); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAdmissionHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdmissionHandler_Handle(t *testing.T) {
	tests := []struct {
		name         string
		body         []byte
		wantResponse ctrlAdmission.Response
		loadStubs    func(*validationmock.MockValidator)
	}{
		{
			name: "default test",
			body: testdata.ValidadmissionBody,
			wantResponse: ctrlAdmission.Response{
				AdmissionResponse: v1.AdmissionResponse{
					Allowed: true,
					Result: &metav1.Status{
						Reason: "",
						Code:   http.StatusOK,
					},
				},
			},
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).Return(&domain.PolicyValidationSummary{}, nil)
			},
		},
		{
			name: "invalid request entity body",
			body: testdata.InvalidadmissionEntity,
			wantResponse: ctrlAdmission.Response{
				AdmissionResponse: v1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Reason: ErrGettingAdmissionEntity,
						Code:   http.StatusInternalServerError,
					},
				},
			},
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0).Return(&domain.PolicyValidationSummary{}, nil)
			},
		},
		{
			name: "ignored namespace test",
			body: testdata.SkippedadmissionBody,
			wantResponse: ctrlAdmission.Response{
				AdmissionResponse: v1.AdmissionResponse{
					Allowed: true,
					Result: &metav1.Status{
						Reason: ExcludedefaultNamespacesMsg,
						Code:   http.StatusOK,
					},
				},
			},
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0).Return(&domain.PolicyValidationSummary{}, nil)
			},
		},
		{
			name: "error during validation",
			body: testdata.ValidadmissionBody,
			wantResponse: ctrlAdmission.Response{
				AdmissionResponse: v1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Reason: ErrValidatingResource,
						Code:   http.StatusInternalServerError,
					},
				},
			},
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).Return(&domain.PolicyValidationSummary{}, fmt.Errorf("validation error"))
			},
		},
		{
			name: "test not allowed",
			body: testdata.ValidadmissionBody,
			wantResponse: ctrlAdmission.Response{
				AdmissionResponse: v1.AdmissionResponse{
					Allowed: false,
					Result: &metav1.Status{
						Reason: "violation",
						Code:   http.StatusForbidden,
					},
				},
			},
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).Return(&domain.PolicyValidationSummary{
					Violations: []domain.PolicyValidation{
						{Message: "violation"},
					},
				}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := require.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			validator := validationmock.NewMockValidator(ctrl)
			tt.loadStubs(validator)
			a := &AdmissionHandler{
				logLevel:  "debug",
				validator: validator,
			}
			var req ctrlAdmission.Request
			err := json.Unmarshal(tt.body, &req)
			assert.Nil(err, "failed to read admission request body test case")
			resp := a.Handle(context.Background(), req)
			assert.Equal(tt.wantResponse, resp, "unexpected admission response")
		})
	}
}
