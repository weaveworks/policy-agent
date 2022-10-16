package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/weaveworks/policy-agent/api/v2beta2"
	pacv2 "github.com/weaveworks/policy-agent/api/v2beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	k8sLabels "k8s.io/apimachinery/pkg/labels"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

	labels := getLabels(obj)
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
	// if err := mgr.GetFieldIndexer().IndexField(
	// 	context.Background(),
	// 	&pacv2.PolicyConfig{},
	// 	"metadata.name",
	// 	func(obj client.Object) []string {
	// 		return []string{obj.GetName()}
	// 	}); err != nil {
	// 	return err
	// }
	return ctrl.NewControllerManagedBy(mgr).
		For(&v2beta2.PolicyConfig{}).
		Complete(pc)
}

func getLabels(config pacv2.PolicyConfig) map[string]string {
	labels := config.Labels
	labels[TargetScopeLabel] = config.Spec.Target.Type()

	if config.Spec.Target.Kind == "" {
		delete(labels, TargetKindLabel)
	} else {
		labels[TargetKindLabel] = config.Spec.Target.Kind
	}

	if config.Spec.Target.Name == "" {
		delete(labels, TargetNameLabel)
	} else {
		labels[TargetNameLabel] = config.Spec.Target.Name
	}

	if config.Spec.Target.Namespace == "" {
		delete(labels, TargetNamespaceLabel)
	} else {
		labels[TargetNamespaceLabel] = config.Spec.Target.Namespace
	}

	if config.Spec.Target.Labels == nil {
		for k := range labels {
			if strings.HasPrefix(k, TargetLablesLabel) {
				delete(labels, k)
			}
		}
	} else {
		for k, v := range config.Spec.Target.Labels {
			labels[fmt.Sprintf("%s.%s", TargetLablesLabel, k)] = v
		}
	}
	return labels
}

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
	if err := config.Validate(); err != nil {
		return admission.Denied(err.Error())
	}

	if err := pc.checkTargetConfict(ctx, *config); err != nil {
		return admission.Denied(err.Error())
	}

	return admission.Allowed("")
}

// InjectDecoder injects the decoder.
func (pc *PolicyConfigValidator) InjectDecoder(d *admission.Decoder) error {
	pc.decoder = d
	return nil
}

func (pc *PolicyConfigValidator) SetupWithManager(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register(
		"/validate-v2beta2-policyconfig",
		&webhook.Admission{Handler: pc},
	)
	return nil
}

func (pc *PolicyConfigValidator) checkTargetConfict(ctx context.Context, config pacv2.PolicyConfig) error {
	var configs pacv2.PolicyConfigList

	labels := make(map[string]string)

	switch config.Spec.Target.Type() {
	case "cluster":
		labels = config.Spec.Target.Labels
		labels[TargetScopeLabel] = "cluster"
	case "namespace":
		labels[TargetNamespaceLabel] = config.Spec.Target.Namespace
		labels[TargetScopeLabel] = "namespace"
	case "resource":
		labels[TargetKindLabel] = config.Spec.Target.Kind
		labels[TargetNameLabel] = config.Spec.Target.Name
		labels[TargetNamespaceLabel] = config.Spec.Target.Namespace
		labels[TargetScopeLabel] = "resource"
	}

	if err := pc.Client.List(ctx, &configs, &client.ListOptions{
		LabelSelector: k8sLabels.SelectorFromSet(labels),
	}); err != nil {
		return fmt.Errorf("failed to list policy configs, error: %v", err)
	}

	if len(configs.Items) > 0 {
		for i := range configs.Items {
			if configs.Items[i].GetName() == config.GetName() {
				continue
			}
			return fmt.Errorf("found policy config '%s' which already targets same targets", configs.Items[i].GetName())
		}
	}
	return nil
}
