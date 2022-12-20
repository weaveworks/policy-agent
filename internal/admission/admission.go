package admission

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/MagalixTechnologies/policy-core/domain"
	"github.com/MagalixTechnologies/policy-core/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlAdmission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// AdmissionHandler listens to admission requests and validates them using a validator
type AdmissionHandler struct {
	logLevel  string
	validator validation.Validator
}

const (
	TypeAdmission               = "Admission"
	DebugLevel                  = "debug"
	ExcludedefaultNamespacesMsg = "default kubernetes namespaces are excluded"
	ErrGettingAdmissionEntity   = "failed to get entity info from admission request"
	ErrValidatingResource       = "failed to validate resource"
)

// NewAdmissionHandler returns an admission handler that listens to k8s validating requests
func NewAdmissionHandler(
	logLevel string,
	validator validation.Validator) *AdmissionHandler {
	return &AdmissionHandler{
		logLevel:  logLevel,
		validator: validator,
	}
}

func (a *AdmissionHandler) handleErrors(err error, errMsg string) ctrlAdmission.Response {
	logger.Errorw("validating admission request error", "error", err, "error-message", errMsg)
	errRsp := ctrlAdmission.ValidationResponse(false, errMsg)
	errRsp.Result.Code = http.StatusInternalServerError
	return errRsp
}

// Handle validates admission requests, implements interface at sigs.k8s.io/controller-runtime/pkg/webhook/admission.Handler
func (a *AdmissionHandler) Handle(ctx context.Context, req ctrlAdmission.Request) ctrlAdmission.Response {
	namespace := req.Namespace
	if namespace == metav1.NamespacePublic || namespace == metav1.NamespaceSystem {
		return ctrlAdmission.ValidationResponse(true, ExcludedefaultNamespacesMsg)
	}

	if a.logLevel == DebugLevel {
		logger.Debugw("admission request body", "payload", req)
	}

	var entitySpec map[string]interface{}
	err := json.Unmarshal(req.Object.Raw, &entitySpec)
	if err != nil {
		return a.handleErrors(err, ErrGettingAdmissionEntity)
	}

	entity := domain.NewEntityFromSpec(entitySpec)
	result, err := a.validator.Validate(ctx, entity, string(req.AdmissionRequest.Operation))
	if err != nil {
		return a.handleErrors(err, ErrValidatingResource)
	}

	if len(result.Violations) > 0 {
		return ctrlAdmission.ValidationResponse(false, generateResponse(result.Violations))
	}

	return ctrlAdmission.ValidationResponse(true, "")
}

// Run starts the admission webhook server
func (a *AdmissionHandler) Run(mgr ctrl.Manager) error {
	webhook := ctrlAdmission.Webhook{Handler: a}
	mgr.GetWebhookServer().Register("/admission", &webhook)

	return nil
}

func generateResponse(violations []domain.PolicyValidation) string {
	var buffer strings.Builder
	for _, violation := range violations {
		buffer.WriteString("==================================================================\n")
		buffer.WriteString(fmt.Sprintf("Policy	: %s\n", violation.Policy.ID))

		if violation.Entity.Namespace == "" {
			buffer.WriteString(fmt.Sprintf("Entity	: %s/%s\n", strings.ToLower(violation.Entity.Kind), violation.Entity.Name))
		} else {
			buffer.WriteString(fmt.Sprintf("Entity	: %s/%s in namespace: %s\n", strings.ToLower(violation.Entity.Kind), violation.Entity.Name, violation.Entity.Namespace))
		}

		buffer.WriteString("Occurrences:\n")
		for _, occurrence := range violation.Occurrences {
			buffer.WriteString(fmt.Sprintf("- %s\n", occurrence.Message))
		}
	}
	return buffer.String()
}
