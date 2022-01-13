package v1

import (
	"context"

	magalixv1 "github.com/MagalixCorp/new-magalix-agent/apiextensions/magalix.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type KubePoliciesClient struct {
	client rest.Interface
}

func NewKubePoliciesClient(c *rest.Config) *KubePoliciesClient {
	config := *c
	config.ContentConfig.GroupVersion = &schema.GroupVersion{Group: magalixv1.GroupName, Version: magalixv1.GroupVersion}
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil
	}

	return &KubePoliciesClient{client: client}
}

func (p *KubePoliciesClient) List(ctx context.Context, opts metav1.ListOptions) (*magalixv1.PolicyList, error) {
	result := magalixv1.PolicyList{}
	err := p.client.
		Get().
		Resource("policies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	return &result, err
}

func (c *KubePoliciesClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*magalixv1.Policy, error) {
	result := magalixv1.Policy{}
	err := c.client.
		Get().
		Resource("policies").
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (p *KubePoliciesClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return p.client.
		Get().
		Resource("policies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}
