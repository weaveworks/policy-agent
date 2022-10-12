package controllers

import (
	"context"

	"github.com/MagalixTechnologies/core/logger"
	"github.com/go-logr/logr"
	"github.com/weaveworks/policy-agent/api/v2beta2"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type PolicyConfigReconciler struct {
	client.Client
	Logger logr.Logger
	Scheme *runtime.Scheme
}

func (pc *PolicyConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger.Infow("============================================================", "name", req.Name, "namespace", req.Namespace)
	return ctrl.Result{}, nil
}

func (pc *PolicyConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(
			predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool {
					return true
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					return true
				},
			},
		).
		For(&v2beta2.Policy{}).
		Complete(pc)
}

// func (pc *PolicyReconciler) listPolicies() (v2beta2.PolicyList, error) {
// 	var policies v2beta2.PolicyList
// 	err := pc.List(context.Background(), &policies)
// 	if err != nil {
// 		return v2beta2.PolicyList{}, err
// 	}
// 	return policies, nil
// }

// func (pc *PolicyReconciler) listPolicySets() (v2beta2.PolicySetList, error) {
// 	var policySets v2beta2.PolicySetList
// 	err := pc.List(context.Background(), &policySets)
// 	if err != nil {
// 		return v2beta2.PolicySetList{}, err
// 	}

// 	// pc.Status().Update()

// 	return policySets, nil
// }
