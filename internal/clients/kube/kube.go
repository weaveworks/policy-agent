package kube

import (
	"context"
	"fmt"
	"strings"

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
	ClientSet       kubernetes.Interface
	DynamicClient   dynamic.Interface
	DiscoveryClient discovery.DiscoveryInterface
}

// NewKubeClient returns a new instance of KubeClient
func NewKubeClient(config *rest.Config) (*KubeClient, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create clientset for kube client, error: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create dynamic client for kube client, error: %w", err)
	}
	return &KubeClient{
		ClientSet:       clientSet,
		DynamicClient:   dynamicClient,
		DiscoveryClient: clientSet.DiscoveryClient}, nil

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

	subjectRules, err := k.ClientSet.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, &rulesSpec, meta.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get agent permissions, error: %w", err)
	}

	return subjectRules, nil
}

// ListResourceItems returns items from a specific reource group version
func (k *KubeClient) ListResourceItems(
	ctx context.Context,
	resource schema.GroupVersionResource,
	namespace string,
	listOptions meta.ListOptions) (*unstructured.UnstructuredList, error) {

	list, err := k.DynamicClient.Resource(resource).Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to list resource %s in namespace %s: %w", resource.Resource, namespace, err)
	}
	return list, nil
}

// GetAPIResources returns all available api resources in the cluster
func (k *KubeClient) GetAPIResources(ctx context.Context) ([]*meta.APIResourceList, error) {
	apiResourcesList, err := k.DiscoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get server api resources: %w", err)
	}
	return apiResourcesList, nil
}

func (k *KubeClient) GetServerVersion() (string, error) {
	version, err := k.DiscoveryClient.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("unable to get cluster version: %w", err)
	}

	return version.String(), nil
}

func (k *KubeClient) GetClusterProvider(ctx context.Context) (string, error) {
	nodes, err := k.ClientSet.CoreV1().Nodes().List(ctx, meta.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("unable to list cluster nodes: %w", err)
	}

	node := nodes.Items[0]
	return strings.Split(node.Spec.ProviderID, ":")[0], nil
}
