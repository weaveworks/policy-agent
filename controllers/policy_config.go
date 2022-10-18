package controllers

import (
	"context"
	"net/http"

	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PolicyConfigValidator struct {
	Client  client.Client
	decoder *admission.Decoder
}

func (pc *PolicyConfigValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	newConfig := &pacv2.PolicyConfig{}
	err := pc.decoder.Decode(req, newConfig)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if err = newConfig.Validate(); err != nil {
		return admission.Denied(err.Error())
	}

	configs := &pacv2.PolicyConfigList{}
	err = pc.Client.List(ctx, configs)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	for _, config := range configs.Items {
		if config.GetName() == newConfig.GetName() {
			continue
		}
		if err := config.CheckTargetOverlap(newConfig); err != nil {
			return admission.Denied(err.Error())
		}
	}
	return admission.Allowed("")
}

func (pc *PolicyConfigValidator) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register(
		"/validate-v2beta2-policyconfig",
		&webhook.Admission{Handler: pc},
	)
	return nil
}

// InjectDecoder injects the decoder.
func (pc *PolicyConfigValidator) InjectDecoder(d *admission.Decoder) error {
	pc.decoder = d
	return nil
}
