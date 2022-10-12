package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/weaveworks/policy-agent/api/v2beta2"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	TargetKindLabel      = fmt.Sprintf("%s/%s", pacv2.GroupVersion.Group, "kind")
	TargetNameLabel      = fmt.Sprintf("%s/%s", pacv2.GroupVersion.Group, "name")
	TargetNamespaceLabel = fmt.Sprintf("%s/%s", pacv2.GroupVersion.Group, "namespace")
	TargetLablesLabel    = fmt.Sprintf("%s/%s", pacv2.GroupVersion.Group, "label")
	TargetScopeLabel     = fmt.Sprintf("%s/%s", pacv2.GroupVersion.Group, "scope")
)

type PolicyConfigReconciler struct {
	client.Client
	Logger logr.Logger
	Scheme *runtime.Scheme
}

func (pc *PolicyConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var obj pacv2.PolicyConfig
	if err := pc.Get(ctx, req.NamespacedName, &obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		pc.Logger.Error(err, "unable to get policy config")
		return ctrl.Result{}, err
	}

	labels := obj.Labels

	if obj.Spec.Target.Kind == "" {
		delete(labels, TargetKindLabel)
	} else {
		labels[TargetKindLabel] = obj.Spec.Target.Kind
	}

	if obj.Spec.Target.Name == "" {
		delete(labels, TargetNameLabel)
	} else {
		labels[TargetNameLabel] = obj.Spec.Target.Name
	}

	if obj.Spec.Target.Namespace == "" {
		delete(labels, TargetNamespaceLabel)
	} else {
		labels[TargetNamespaceLabel] = obj.Spec.Target.Namespace
	}

	if obj.Spec.Target.Labels == nil {
		for k := range labels {
			if strings.HasPrefix(k, TargetLablesLabel) {
				delete(labels, k)
			}
		}
	} else {
		for k, v := range obj.Spec.Target.Labels {
			labels[fmt.Sprintf("%s/%s", TargetLablesLabel, k)] = v
		}
	}

	labels[TargetScopeLabel] = obj.Spec.Target.Type()

	obj.SetLabels(labels)

	if err := pc.Update(ctx, &obj); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{Requeue: true}, nil
		}
		pc.Logger.Error(err, "unable to update policy config")
		return ctrl.Result{}, err
	}

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
		For(&v2beta2.PolicyConfig{}).
		Complete(pc)
}

// func (pc *PolicyConfigReconciler) labelInjector(obj pacv2.PolicyConfig) client.Object {

// 	return obj
// }

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

// prefix/labels/key: value
