package mutation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/weaveworks/weave-policy-agent/pkg/logger"
	"github.com/weaveworks/weave-policy-agent/pkg/policy-core/domain"
	"github.com/weaveworks/weave-policy-agent/pkg/policy-core/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlAdmission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type MutationHandler struct {
	validator validation.Validator
}

func NewMutationHandler(validator validation.Validator) *MutationHandler {
	return &MutationHandler{
		validator: validator,
	}
}

func (m *MutationHandler) handleErrors(err error, errMsg string) ctrlAdmission.Response {
	logger.Errorw("validating mutation request error", "error", err, "error-message", errMsg)
	errRsp := ctrlAdmission.ValidationResponse(false, errMsg)
	errRsp.Result.Code = http.StatusInternalServerError
	return errRsp
}

func (m *MutationHandler) Handle(ctx context.Context, req ctrlAdmission.Request) ctrlAdmission.Response {
	if req.Namespace == metav1.NamespacePublic || req.Namespace == metav1.NamespaceSystem {
		return ctrlAdmission.ValidationResponse(true, "exclude default system namespaces")
	}

	var entitySpec map[string]interface{}
	err := json.Unmarshal(req.Object.Raw, &entitySpec)
	if err != nil {
		return m.handleErrors(err, fmt.Sprintf("failed to unmarshal entity %s/%s spec into a map", req.Namespace, req.Name))
	}

	entity := domain.NewEntityFromSpec(entitySpec)
	result, err := m.validator.Validate(ctx, entity, string(req.AdmissionRequest.Operation))
	if err != nil {
		return m.handleErrors(err, fmt.Sprintf("failed to validate entity %s/%s", req.Namespace, req.Name))
	}

	if result.Mutation != nil {
		logger.Infow("mutating resource", "name", req.Name, "namespace", req.Namespace)
		mutated, err := result.Mutation.NewResource()
		if err != nil {
			return m.handleErrors(err, fmt.Sprintf("failed to mutate entity %s/%s ", req.Namespace, req.Name))
		}
		return ctrlAdmission.PatchResponseFromRaw(result.Mutation.OldResource(), mutated)
	}

	return ctrlAdmission.Allowed("")
}

// Run starts the mutation webhook server
func (m *MutationHandler) Run(mgr ctrl.Manager) error {
	webhook := ctrlAdmission.Webhook{Handler: m}
	mgr.GetWebhookServer().Register("/mutation", &webhook)
	return nil
}
