package k8s

import (
	"context"
	"fmt"
	"testing"

	"github.com/MagalixCorp/magalix-policy-agent/clients/kube"
	"github.com/MagalixCorp/magalix-policy-agent/pkg/domain"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	discoverfake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

type DisoceryMock struct {
	discoverfake.FakeDiscovery
	ApiList []*meta.APIResourceList
	err     error
}

func (d *DisoceryMock) ServerPreferredResources() ([]*meta.APIResourceList, error) {
	if d.err != nil {
		return nil, d.err
	}
	return d.ApiList, nil

}

type Permissions struct {
	review *authv1.SelfSubjectRulesReview
	err    error
}

func TestGetEntitiesSources(t *testing.T) {
	type args struct {
		permissions     Permissions
		dynamicClient   dynamic.Interface
		discoveryClient discovery.DiscoveryInterface
	}
	tests := []struct {
		name    string
		args    args
		want    []domain.EntitiesSource
		wantErr bool
	}{
		{
			name: "standard test",
			args: args{
				permissions: Permissions{
					review: &authv1.SelfSubjectRulesReview{
						Status: authv1.SubjectRulesReviewStatus{
							ResourceRules: []authv1.ResourceRule{
								{
									Verbs:     []string{"list"},
									Resources: []string{"deployments"},
								},
								{
									Verbs:     []string{"list"},
									Resources: []string{"replicasets"},
								},
							},
						},
					},
				},
				discoveryClient: &DisoceryMock{
					ApiList: []*meta.APIResourceList{
						{
							GroupVersion: "apps/v1",
							APIResources: []meta.APIResource{
								{
									Name:  "deployments",
									Kind:  "Deployment",
									Verbs: []string{"watch", "create", "list"},
								},
							},
						},
						{
							GroupVersion: "apps/v1",
							APIResources: []meta.APIResource{
								{
									Name:  "replicasets",
									Kind:  "ReplicaSet",
									Verbs: []string{"watch", "create", "list"},
								},
							},
						},
					},
				},
			},
			want: []domain.EntitiesSource{
				&K8SEntitySource{
					resource: schema.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "deployments",
					},
					kind: "Deployment",
				},
				&K8SEntitySource{
					resource: schema.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "replicasets",
					},
					kind: "ReplicaSet",
				},
			},
		},
		{
			name: "error getting permissions",
			args: args{
				permissions: Permissions{
					err: fmt.Errorf("error"),
				},
				discoveryClient: &DisoceryMock{},
			},
			wantErr: true,
		},
		{
			name: "*  verbs in permission",
			args: args{
				permissions: Permissions{
					review: &authv1.SelfSubjectRulesReview{
						Status: authv1.SubjectRulesReviewStatus{
							ResourceRules: []authv1.ResourceRule{
								{
									Verbs:     []string{"*"},
									Resources: []string{"deployments"},
								},
							},
						},
					},
				},
				discoveryClient: &DisoceryMock{
					ApiList: []*meta.APIResourceList{
						{
							GroupVersion: "apps/v1",
							APIResources: []meta.APIResource{
								{
									Name:  "deployments",
									Kind:  "Deployment",
									Verbs: []string{"watch", "create", "list"},
								},
							},
						},
					},
				},
			},
			want: []domain.EntitiesSource{
				&K8SEntitySource{
					resource: schema.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "deployments",
					},
					kind: "Deployment",
				},
			},
		},
		{
			name: "* reousces permissions",
			args: args{
				permissions: Permissions{
					review: &authv1.SelfSubjectRulesReview{
						Status: authv1.SubjectRulesReviewStatus{
							ResourceRules: []authv1.ResourceRule{
								{
									Verbs:     []string{"list"},
									Resources: []string{"*"},
								},
							},
						},
					},
				},
				discoveryClient: &DisoceryMock{
					ApiList: []*meta.APIResourceList{
						{
							GroupVersion: "apps/v1",
							APIResources: []meta.APIResource{
								{
									Name:  "deployments",
									Kind:  "Deployment",
									Verbs: []string{"watch", "create", "list"},
								},
							},
						},
					},
				},
			},
			want: []domain.EntitiesSource{
				&K8SEntitySource{
					resource: schema.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "deployments",
					},
					kind: "Deployment",
				},
			},
		},
		{
			name: "error getting api resources",
			args: args{
				permissions: Permissions{
					review: &authv1.SelfSubjectRulesReview{
						Status: authv1.SubjectRulesReviewStatus{
							ResourceRules: []authv1.ResourceRule{
								{
									Verbs:     []string{"list"},
									Resources: []string{"*"},
								},
							},
						},
					},
				},
				discoveryClient: &DisoceryMock{
					err: fmt.Errorf("expected error"),
				},
			},
			wantErr: true,
		},
		{
			name: "skip resource without list permissions",
			args: args{
				permissions: Permissions{
					review: &authv1.SelfSubjectRulesReview{
						Status: authv1.SubjectRulesReviewStatus{
							ResourceRules: []authv1.ResourceRule{
								{
									Verbs:     []string{"list"},
									Resources: []string{"*"},
								},
							},
						},
					},
				},
				discoveryClient: &DisoceryMock{
					ApiList: []*meta.APIResourceList{
						{
							GroupVersion: "apps/v1",
							APIResources: []meta.APIResource{
								{
									Name:  "localsubjectaccessreviews",
									Kind:  "LocalSubjectAccessReview",
									Verbs: []string{"create"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "skip magalix policies",
			args: args{
				permissions: Permissions{
					review: &authv1.SelfSubjectRulesReview{
						Status: authv1.SubjectRulesReviewStatus{
							ResourceRules: []authv1.ResourceRule{
								{
									Verbs:     []string{"list"},
									Resources: []string{"*"},
								},
							},
						},
					},
				},
				discoveryClient: &DisoceryMock{
					ApiList: []*meta.APIResourceList{
						{
							GroupVersion: "magalix.com/v1",
							APIResources: []meta.APIResource{
								{
									Name:  "policies",
									Kind:  "Policies",
									Verbs: []string{"watch", "create", "list"},
								},
							},
						},
						{
							GroupVersion: "apps/v1",
							APIResources: []meta.APIResource{
								{
									Name:  "deployments",
									Kind:  "Deployment",
									Verbs: []string{"watch", "create", "list"},
								},
							},
						},
					},
				},
			},
			want: []domain.EntitiesSource{
				&K8SEntitySource{
					resource: schema.GroupVersionResource{
						Group:    "apps",
						Version:  "v1",
						Resource: "deployments",
					},
					kind: "Deployment",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := require.New(t)
			ctx := context.Background()
			cli := fake.NewSimpleClientset()
			cli.PrependReactor("create", "selfsubjectrulesreviews", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				if test.args.permissions.err != nil {
					return true, nil, test.args.permissions.err
				}
				return true, test.args.permissions.review, nil
			})
			kubeClient := kube.NewKubeClientByArgs(cli, test.args.dynamicClient, test.args.discoveryClient)
			gotSources, err := GetEntitiesSources(ctx, kubeClient)
			assert.Equal(test.wantErr, err != nil, "unexpected error result")
			assert.Equal(len(test.want), len(gotSources), "unexpected entities sources number")

			for i := range test.want {
				gotSource, ok := gotSources[i].(*K8SEntitySource)
				assert.True(ok)
				wantSource, ok := test.want[i].(*K8SEntitySource)
				assert.True(ok)
				assert.Equal(wantSource.resource, gotSource.resource, "unexpected entity source resource")
				assert.Equal(wantSource.kind, gotSource.kind, "unexpected entity source kind")
			}
		})
	}
}

type Reaction struct {
	objects []unstructured.Unstructured
	err     error
}

func TestK8SEntitySource_List(t *testing.T) {
	type fields struct {
		resource schema.GroupVersionResource
		kind     string
	}
	type args struct {
		listOptions *domain.ListOptions
	}
	type want struct {
		data    []unstructured.Unstructured
		hasNext bool
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		reaction Reaction
		want     want
		wantErr  bool
	}{
		{
			name: "standard test",
			fields: fields{
				resource: schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"},
				kind:     "Deployment",
			},
			args: args{
				listOptions: &domain.ListOptions{},
			},
			reaction: Reaction{
				objects: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "Deployment",
							"metadata": map[string]interface{}{
								"namespace": "default",
								"name":      "test",
							},
						},
					},
				},
			},
			want: want{
				data: []unstructured.Unstructured{
					{
						Object: map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "Deployment",
							"metadata": map[string]interface{}{
								"namespace": "default",
								"name":      "test",
							},
						},
					},
				},
				hasNext: false,
			},
		},
		{
			name: "error while performing list call",
			fields: fields{
				resource: schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"},
				kind:     "Deployment",
			},
			args: args{
				listOptions: &domain.ListOptions{},
			},
			reaction: Reaction{
				err: fmt.Errorf("expected error"),
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			assert := require.New(t)

			scheme.AddKnownTypeWithName(schema.GroupVersionKind{Kind: "DeploymentList", Version: "v1", Group: "apps"}, &unstructured.UnstructuredList{})
			dynamicCli := dynamicfake.NewSimpleDynamicClient(scheme)
			cli := fake.NewSimpleClientset()
			kubeClient := kube.NewKubeClientByArgs(cli, dynamicCli, &DisoceryMock{})
			dynamicCli.PrependReactor("list", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				if test.reaction.err != nil {
					return true, nil, test.reaction.err
				}
				return true, &unstructured.UnstructuredList{
					Items: test.reaction.objects}, nil
			})
			k := &K8SEntitySource{
				resource:   test.fields.resource,
				kubeClient: kubeClient,
				kind:       test.fields.kind,
			}
			ctx := context.Background()
			got, err := k.List(ctx, test.args.listOptions)
			if test.wantErr {
				assert.Equal(test.wantErr, err != nil, "unexpected error result")
				return
			}
			wantList := &domain.EntitiesList{HasNext: test.want.hasNext}
			for i := range test.want.data {
				wantList.Data = append(wantList.Data, domain.NewEntityFromSpec(test.want.data[i].Object))
			}
			assert.Equal(wantList, got, "unexpected entities list value")
		})
	}
}

func TestK8SEntitySource_Kind(t *testing.T) {
	type fields struct {
		resource schema.GroupVersionResource
		kind     string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "standard test",
			fields: fields{
				resource: schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"},
				kind:     "Deployment",
			},
			want: "Deployment",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := require.New(t)

			dynamicCli := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
			cli := fake.NewSimpleClientset()
			kubeClient := kube.NewKubeClientByArgs(cli, dynamicCli, &DisoceryMock{})
			k := &K8SEntitySource{
				resource:   test.fields.resource,
				kubeClient: kubeClient,
				kind:       test.fields.kind,
			}
			got := k.Kind()
			assert.Equal(test.want, got, "unexpedted kind")
		})
	}
}
