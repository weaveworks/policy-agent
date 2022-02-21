package kube

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authorization/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubeClient provides interface to various k8s api calls
type KubeClient struct {
	clientSet       kubernetes.Interface
	dynamicClient   dynamic.Interface
	discoveryClient discovery.DiscoveryInterface
}

// NewKubeClientByArgs return KubeClient by providing the needed interfaces
func NewKubeClientByArgs(
	clientSet kubernetes.Interface,
	dynamicClient dynamic.Interface,
	discoveryClient discovery.DiscoveryInterface,
) *KubeClient {
	return &KubeClient{
		clientSet:       clientSet,
		dynamicClient:   dynamicClient,
		discoveryClient: discoveryClient}
}

// NewKubeClientByConfig returns a new instance of KubeClient
func NewKubeClientByConfig(config *rest.Config) (*KubeClient, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create clientset for kube client, error: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create dynamic client for kube client, error: %w", err)
	}
	return NewKubeClientByArgs(clientSet, dynamicClient, clientSet.DiscoveryClient), nil

}

// GetAgentPermissions retrieves allowed permissions for the agent
func (k *KubeClient) GetAgentPermissions(ctx context.Context) (*authv1.SelfSubjectRulesReview, error) {

	rulesSpec := authv1.SelfSubjectRulesReview{
		Spec: authv1.SelfSubjectRulesReviewSpec{
			Namespace: "kube-system",
		},
		Status: authv1.SubjectRulesReviewStatus{
			Incomplete: false,
		},
	}

	subjectRules, err := k.clientSet.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, &rulesSpec, meta.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get agent permissions, error: %w", err)
	}

	return subjectRules, nil
}

// List returns data from a specific reource group version
func (k *KubeClient) List(
	ctx context.Context,
	resource schema.GroupVersionResource,
	namespace string,
	listOptions meta.ListOptions) (*unstructured.UnstructuredList, error) {

	list, err := k.dynamicClient.Resource(resource).Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to list resource %s, %w", resource.Resource, err)
	}
	return list, nil
}

// GetAPIResources returns all available api resources in the cluster
func (k *KubeClient) GetAPIResources(ctx context.Context) ([]*meta.APIResourceList, error) {
	apiResourcesList, err := k.discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get server api resources, %w", err)
	}
	return apiResourcesList, nil
}
