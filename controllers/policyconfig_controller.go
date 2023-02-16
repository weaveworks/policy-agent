package controllers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MagalixTechnologies/core/logger"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PolicyConfigController struct {
	Client  client.Client
	decoder *admission.Decoder
}

const (
	policyConfigIndexKey = "spec.config.policies"
)

func checkTargetOverlap(config, newConfig pacv2.PolicyConfig) error {
	if config.Spec.Match.Namespaces != nil {
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

func (pc *PolicyConfigController) Handle(ctx context.Context, req admission.Request) admission.Response {
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

func (pc *PolicyConfigController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger.Infow("reconciling policy config", "policy config", req.Name)

	policyConfig := pacv2.PolicyConfig{}
	if err := pc.Client.Get(ctx, req.NamespacedName, &policyConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !policyConfig.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	policyConfigConfig := policyConfig.Spec.Config
	policiesIDs := []string{}

	for policyID := range policyConfigConfig {
		policiesIDs = append(policiesIDs, policyID)
	}

	missingPolicies := []string{}

	for _, policyID := range policiesIDs {
		policy := pacv2.Policy{}
		policyName := types.NamespacedName{
			Name: policyID,
		}
		if err := pc.Client.Get(ctx, policyName, &policy); err != nil {
			if apierrors.IsNotFound(err) {
				missingPolicies = append(missingPolicies, policyID)
			} else {
				return ctrl.Result{}, err
			}
		}
	}
	patch := client.MergeFrom(policyConfig.DeepCopy())
	policyConfig.SetPolicyConfigStatus(missingPolicies)

	logger.Infow("updating policy config config status", "name", req.Name, "status", policyConfig.Status.Status, "warnings", missingPolicies)
	if err := pc.Client.Patch(ctx, &policyConfig, patch); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (p *PolicyConfigController) reconcile(obj client.Object) []reconcile.Request {
	policiesConfigs := &pacv2.PolicyConfigList{}
	opts := client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(policyConfigIndexKey, obj.GetName()),
	}

	err := p.Client.List(context.Background(), policiesConfigs, &opts)
	if err != nil {
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, len(policiesConfigs.Items))
	for i, item := range policiesConfigs.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: item.Name,
			},
		}
	}
	return requests
}

func (pc *PolicyConfigController) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register(
		"/validate-v2beta2-policyconfig",
		&webhook.Admission{Handler: pc},
	)

	err := mgr.GetFieldIndexer().IndexField(context.Background(), &pacv2.PolicyConfig{}, policyConfigIndexKey, func(obj client.Object) []string {
		policyConfig, ok := obj.(*pacv2.PolicyConfig)
		if !ok {
			return nil
		}
		policyIDS := []string{}
		for policyID := range policyConfig.Spec.Config {
			policyIDS = append(policyIDS, policyID)
		}
		return policyIDS
	})

	if err != nil {
		return err
	}

	// watch both policies and policy config in case user changed either of them
	return ctrl.NewControllerManagedBy(mgr).
		For(&pacv2.PolicyConfig{}).
		Watches(
			&source.Kind{Type: &pacv2.Policy{}},
			handler.EnqueueRequestsFromMapFunc(pc.reconcile),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(pc)

}

// InjectDecoder injects the decoder.
func (pc *PolicyConfigController) InjectDecoder(d *admission.Decoder) error {
	pc.decoder = d
	return nil
}
