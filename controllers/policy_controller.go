package controllers

import (
	"context"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/weaveworks/policy-agent/api/v2beta2"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	modeProviderMap = map[string]string{
		pacv2.PolicySetAuditMode:       pacv2.PolicyKubernetesProvider,
		pacv2.PolicySetAdmissionMode:   pacv2.PolicyKubernetesProvider,
		pacv2.PolicySetTFAdmissionMode: pacv2.PolicyTerraformProvider,
	}
)

type PolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile policy reconcile handler
func (p *PolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger.Infow("reconciling policy", "policy", req.Name)

	var policy pacv2.Policy
	if err := p.Get(ctx, req.NamespacedName, &policy); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !policy.ObjectMeta.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	var policySets pacv2.PolicySetList
	if err := p.List(ctx, &policySets); err != nil {
		return ctrl.Result{}, err
	}

	modes := map[string]struct {
		count   int  // total policy sets of this mode
		matched bool // set to true if the policy matches any policy set of this mode
	}{
		pacv2.PolicySetAuditMode:       {},
		pacv2.PolicySetAdmissionMode:   {},
		pacv2.PolicySetTFAdmissionMode: {},
	}

	for _, policySet := range policySets.Items {
		mode := modes[policySet.Spec.Mode]
		if policySet.Match(policy) {
			mode.matched = true
		}
		mode.count++
		modes[policySet.Spec.Mode] = mode
	}

	var policyModes []string
	for name, mode := range modes {
		if modeProviderMap[name] == policy.Spec.Provider && (mode.matched || mode.count == 0) {
			policyModes = append(policyModes, name)
		}
	}

	patch := client.MergeFrom(policy.DeepCopy())
	policy.SetModeLabels(policyModes)

	if err := p.Patch(ctx, &policy, patch); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	patch = client.MergeFrom(policy.DeepCopy())
	policy.Status.SetModes(policyModes)

	logger.Infow("updating policy status.modes", "policy", req.Name, "modes", policy.Status.ModesString)

	if err := p.Status().Patch(ctx, &policy, patch); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (p *PolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v2beta2.Policy{}).
		Watches(
			&source.Kind{Type: &pacv2.PolicySet{}},
			handler.EnqueueRequestsFromMapFunc(p.reconcile),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(p)
}

func (p *PolicyReconciler) reconcile(_ client.Object) []reconcile.Request {
	policies := &pacv2.PolicyList{}
	err := p.List(context.Background(), policies)
	if err != nil {
		return []reconcile.Request{}
	}
	requests := make([]reconcile.Request, len(policies.Items))
	for i, item := range policies.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: item.Name,
			},
		}
	}
	return requests
}
