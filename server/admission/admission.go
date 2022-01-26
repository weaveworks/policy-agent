package admission

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/validation"
	"github.com/MagalixTechnologies/core/logger"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type AdmissionWatcher func(ctx context.Context, reviewRequest v1.AdmissionReview) (*v1.AdmissionReview, error)

var universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()

type AdmissionHandler struct {
	address   string
	certFile  string
	keyFile   string
	logLevel  string
	validator validation.Validator
}

const (
	SourceAdmission = "Admission"
	DebugLevel      = "debug"
)

func NewAdmissionHandler(
	address string,
	certFile string,
	keyFile string,
	logLevel string,
	validator validation.Validator) *AdmissionHandler {
	return &AdmissionHandler{
		address:   address,
		certFile:  certFile,
		keyFile:   keyFile,
		logLevel:  logLevel,
		validator: validator,
	}
}

func writeResponse(writer http.ResponseWriter, v interface{}, status int) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	err := enc.Encode(v)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Error while writing response, error: %s", err)
		writer.Write([]byte(msg))
	}
	writer.WriteHeader(status)
	writer.Write(buf.Bytes())
}

// Register regsiters a function to handle admission requests
func (a *AdmissionHandler) Register(admissionFunc AdmissionWatcher) http.HandlerFunc {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			msg := "unexpected error while reading request body"
			logger.Warn(msg)
			writeResponse(writer, msg, http.StatusInternalServerError)
			return
		}

		if a.logLevel == DebugLevel {
			var payload map[string]interface{}
			json.Unmarshal(body, &payload)
			logger.Debugw("admission request", "payload", payload)
		}
		var reviewRequest v1.AdmissionReview
		_, _, err = universalDeserializer.Decode(body, nil, &reviewRequest)

		if err != nil || reviewRequest.Request == nil {
			msg := "received incorrect admission request"
			logger.Warn(msg)
			writeResponse(writer, msg, http.StatusInternalServerError)
			return
		}

		reviewResponse, err := admissionFunc(request.Context(), reviewRequest)
		if err != nil {
			logger.Errorf("validating admission request error", "error", err)
			writeResponse(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writeResponse(writer, reviewResponse, http.StatusOK)
	})
}

// ValidateRequest include logic to validate admission requests
func (a *AdmissionHandler) ValidateRequest(ctx context.Context, reviewRequest v1.AdmissionReview) (*v1.AdmissionReview, error) {

	reviewResponse := v1.AdmissionReview{
		Response: &v1.AdmissionResponse{
			UID:     reviewRequest.Request.UID,
			Allowed: true,
		},
	}
	reviewResponse.APIVersion = reviewRequest.APIVersion
	reviewResponse.Kind = "AdmissionReview"

	namespace := reviewRequest.Request.Namespace
	if namespace == metav1.NamespacePublic || namespace == metav1.NamespaceSystem {
		return &reviewResponse, nil
	}

	var entitySpec map[string]interface{}
	err := json.Unmarshal(reviewRequest.Request.Object.Raw, &entitySpec)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity info from admission request")
	}

	entity := domain.NewEntityBySpec(entitySpec)
	result, err := a.validator.Validate(ctx, entity, SourceAdmission)
	if err != nil {
		return nil, err
	}

	if len(result.Violations) > 0 {
		violationsMessages := result.GetViolationMessages()
		reviewResponse.Response.Allowed = false
		reviewResponse.Response.Result = &metav1.Status{
			Message: strings.Join(violationsMessages, "\n"),
		}
	}

	return &reviewResponse, nil
}

// Run start the admission webhook server
func (a *AdmissionHandler) Run(ctx context.Context) error {
	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error {
		mux := http.NewServeMux()
		mux.Handle("/admission", a.Register(a.ValidateRequest))
		server := &http.Server{
			Addr:    a.address,
			Handler: mux,
		}
		return server.ListenAndServeTLS(a.certFile, a.keyFile)
	})
	return eg.Wait()
}
