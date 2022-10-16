package controllers

import (
	"context"
	"fmt"
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
	config := &pacv2.PolicyConfig{}
	err := pc.decoder.Decode(req, config)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if config.Spec.Match == nil {
		configs := &pacv2.PolicyConfigList{}
		err = pc.Client.List(ctx, configs, &client.ListOptions{Namespace: config.Namespace})
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		for i := range configs.Items {
			if configs.Items[i].Spec.Match == nil && configs.Items[i].GetName() != config.GetName() {
				return admission.Denied(fmt.Sprintf(
					"failed to create policy config '%s'. namespace '%s' already has policy config '%s' with namespace scope ",
					config.GetName(),
					config.GetName(),
					configs.Items[i].GetName(),
				))
			}
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
