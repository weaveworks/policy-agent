package terraform

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/weaveworks/weave-policy-agent/pkg/logger"
	"github.com/weaveworks/weave-policy-agent/pkg/policy-core/domain"
	"github.com/weaveworks/weave-policy-agent/pkg/policy-core/validation"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	TypeTFAdmission = "TFAdmission"
)

type Response struct {
	Passed     bool                      `json:"passed"`
	Violations []domain.PolicyValidation `json:"violations"`
}

// TerraformHandler listens to terraform validation requests and validates them using a validator
type TerraformHandler struct {
	logLevel  string
	validator validation.Validator
}

// NewTerraformHandler returns an terraform validation handler that listens to terraform validating requests
func NewTerraformHandler(logLevel string, validator validation.Validator) *TerraformHandler {
	return &TerraformHandler{
		logLevel:  logLevel,
		validator: validator,
	}
}

// Handle validates terraform validation requests
func (a *TerraformHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	var entitySpec map[string]interface{}
	err := json.NewDecoder(req.Body).Decode(&entitySpec)
	if err != nil {
		http.Error(
			rw,
			fmt.Sprintf("invalid request body, error: %v", err),
			http.StatusBadRequest,
		)
		return
	}

	entity := domain.NewEntityFromSpec(entitySpec)
	logger.Infow("received valid request", "namespace", entity.Namespace, "name", entity.Name)

	result, err := a.validator.Validate(req.Context(), entity, "terraform")
	if err != nil {
		http.Error(
			rw,
			fmt.Sprintf("failed to validate resource, error: %v", err),
			http.StatusInternalServerError,
		)
		return
	}

	var response Response
	if len(result.Violations) > 0 {
		for i := range result.Violations {
			if _, ok := result.Violations[i].Entity.Manifest["status"]; ok {
				result.Violations[i].Entity.Manifest["status"] = nil
			}
		}
		response.Violations = result.Violations
	} else {
		response.Passed = true
	}

	logger.Infow(
		"resource is validated",
		"namespace", entity.Namespace,
		"name", entity.Name,
		"passed",
		response.Passed,
		"violations",
		len(response.Violations),
	)

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	json.NewEncoder(rw).Encode(response)
}

// Run starts the webhook server
func (a *TerraformHandler) Run(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register("/terraform/admission", a)
	return nil
}
