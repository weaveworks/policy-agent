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

func checkTargetOverlap(config, newConfig pacv2.PolicyConfig) error {
	if config.Spec.Match.Workspaces != nil {
		if newConfig.Spec.Match.Workspaces == nil {
			return nil
		}
		workspaces := map[string]struct{}{}
		for _, workspace := range config.Spec.Match.Workspaces {
			workspaces[workspace] = struct{}{}
		}

		for _, workspace := range newConfig.Spec.Match.Workspaces {
			if _, ok := workspaces[workspace]; ok {
				return fmt.Errorf("policy config '%s' already targets workspace '%s'", config.GetName(), workspace)
			}
		}

	} else if config.Spec.Match.Namespaces != nil {
		if newConfig.Spec.Match.Namespaces == nil {
			return nil
		}
		namespaces := map[string]struct{}{}
		for _, namespace := range config.Spec.Match.Namespaces {
			namespaces[namespace] = struct{}{}
		}

		for _, namespace := range newConfig.Spec.Match.Namespaces {
			if _, ok := namespaces[namespace]; ok {
				return fmt.Errorf("policy config '%s' already targets namespace '%s'", config.GetName(), namespace)
			}
		}
	} else if config.Spec.Match.Applications != nil {
		if newConfig.Spec.Match.Applications == nil {
			return nil
		}
		apps := map[string]struct{}{}
		for _, app := range config.Spec.Match.Applications {
			apps[app.ID()] = struct{}{}
		}

		for _, app := range newConfig.Spec.Match.Applications {
			if _, ok := apps[app.ID()]; ok {
				return fmt.Errorf("policy config '%s' already targets application '%s'", config.GetName(), app.ID())
			}
		}
	} else if config.Spec.Match.Resources != nil {
		if newConfig.Spec.Match.Resources == nil {
			return nil
		}
		resources := map[string]struct{}{}
		for _, resource := range config.Spec.Match.Resources {
			resources[resource.ID()] = struct{}{}
		}

		for _, resource := range newConfig.Spec.Match.Resources {
			if _, ok := resources[resource.ID()]; ok {
				return fmt.Errorf("policy config '%s' already targets resource '%s'", config.GetName(), resource.ID())
			}
		}
	}
	return nil
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
		// in case of update event, skip new config
		if config.GetName() == newConfig.GetName() {
			continue
		}
		if err := checkTargetOverlap(config, *newConfig); err != nil {
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
