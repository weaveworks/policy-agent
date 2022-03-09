package admission

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/validation"
	validationmock "github.com/MagalixCorp/magalix-policy-agent/pkg/validation/mock"
	"github.com/MagalixCorp/magalix-policy-agent/server/admission/testdata"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewAdmissionHandler(t *testing.T) {
	type args struct {
		address   string
		certFile  string
		keyFile   string
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
				address:   ":9000",
				certFile:  "file.crt",
				keyFile:   "file.key",
				logLevel:  "info",
				validator: validator,
			},
			want: &AdmissionHandler{
				address:   ":9000",
				certFile:  "file.crt",
				keyFile:   "file.key",
				logLevel:  "info",
				validator: validator,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAdmissionHandler(tt.args.address, tt.args.certFile, tt.args.keyFile, tt.args.logLevel, tt.args.validator); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAdmissionHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdmissionHandler_Validate_Request(t *testing.T) {
	type fields struct {
		address  string
		certFile string
		keyFile  string
		logLevel string
	}
	tests := []struct {
		name           string
		fields         fields
		body           io.Reader
		wantResponse   *v1.AdmissionReview
		wantStatusCode int
		wantErr        string
		loadStubs      func(*validationmock.MockValidator)
	}{
		{
			name: "default test",
			fields: fields{
				address:  ":9000",
				certFile: "file.crt",
				keyFile:  "file.key",
				logLevel: "debug",
			},
			body: testdata.GetReader("valid"),
			wantResponse: &v1.AdmissionReview{
				Response: &v1.AdmissionResponse{
					UID:     "705ab4f5-6393-11e8-b7cc-42010a800002",
					Allowed: true,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "AdmissionReview",
					APIVersion: "admission.k8s.io/v1",
				},
			},
			wantStatusCode: http.StatusOK,
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).Return(&domain.PolicyValidationSummary{}, nil)
			},
		},
		{
			name: "invalid request body",
			fields: fields{
				address:  ":9000",
				certFile: "file.crt",
				keyFile:  "file.key",
				logLevel: "debug",
			},
			body:           testdata.GetReader("error"),
			wantResponse:   &v1.AdmissionReview{},
			wantStatusCode: http.StatusInternalServerError,
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0).Return(&domain.PolicyValidationSummary{}, nil)
			},
			wantErr: invalidRequestBody,
		},
		{
			name: "invalid admission request body",
			fields: fields{
				address:  ":9000",
				certFile: "file.crt",
				keyFile:  "file.key",
				logLevel: "debug",
			},
			body:           testdata.GetReader("invalid"),
			wantResponse:   &v1.AdmissionReview{},
			wantStatusCode: http.StatusInternalServerError,
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0).Return(&domain.PolicyValidationSummary{}, nil)
			},
			wantErr: invalidadmissionRequest,
		},
		{
			name: "ignored namespace test",
			fields: fields{
				address:  ":9000",
				certFile: "file.crt",
				keyFile:  "file.key",
				logLevel: "debug",
			},
			body: testdata.GetReader("skip"),
			wantResponse: &v1.AdmissionReview{
				Response: &v1.AdmissionResponse{
					UID:     "705ab4f5-6393-11e8-b7cc-42010a800002",
					Allowed: true,
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "AdmissionReview",
					APIVersion: "admission.k8s.io/v1",
				},
			},
			wantStatusCode: http.StatusOK,
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0).Return(&domain.PolicyValidationSummary{}, nil)
			},
		},
		{
			name: "error during validation",
			fields: fields{
				address:  ":9000",
				certFile: "file.crt",
				keyFile:  "file.key",
				logLevel: "debug",
			},
			body:           testdata.GetReader("valid"),
			wantResponse:   &v1.AdmissionReview{},
			wantStatusCode: http.StatusInternalServerError,
			loadStubs: func(val *validationmock.MockValidator) {
				val.EXPECT().Validate(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(1).Return(&domain.PolicyValidationSummary{}, fmt.Errorf("validation error"))
			},
		},
		{
			name: "test not allowed",
			fields: fields{
				address:  ":9000",
				certFile: "file.crt",
				keyFile:  "file.key",
				logLevel: "debug",
			},
			body: testdata.GetReader("valid"),
			wantResponse: &v1.AdmissionReview{
				Response: &v1.AdmissionResponse{
					UID:     "705ab4f5-6393-11e8-b7cc-42010a800002",
					Allowed: false,
					Result: &metav1.Status{
						Message: "violation",
					},
				},
				TypeMeta: metav1.TypeMeta{
					Kind:       "AdmissionReview",
					APIVersion: "admission.k8s.io/v1",
				},
			},
			wantStatusCode: http.StatusOK,
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
				address:   tt.fields.address,
				certFile:  tt.fields.certFile,
				keyFile:   tt.fields.keyFile,
				logLevel:  tt.fields.logLevel,
				validator: validator,
			}
			handler := a.Register(a.ValidateRequest)
			req := httptest.NewRequest(http.MethodPost, "/admission", tt.body)
			w := httptest.NewRecorder()
			handler(w, req)
			res := w.Result()
			defer res.Body.Close()
			assert.Contains([]int{
				http.StatusOK,
				http.StatusInternalServerError,
			}, res.StatusCode)

			if tt.wantStatusCode != http.StatusOK {
				var gotResponse string
				err := json.NewDecoder(res.Body).Decode(&gotResponse)
				assert.Nil(err, "error while decoding admission handler error response, %s", err)
				if tt.wantErr != "" {
					assert.Equal(tt.wantErr, gotResponse)
				}
			} else {
				var gotResponse v1.AdmissionReview
				err := json.NewDecoder(res.Body).Decode(&gotResponse)
				assert.Empty(err, "error while decoding admission handler response, %s", err)
				assert.Equal(tt.wantResponse, &gotResponse)
			}
		})
	}
}
