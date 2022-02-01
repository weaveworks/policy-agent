package v1

import (
	"context"
	"fmt"
	"time"

	magalixv1 "github.com/MagalixCorp/magalix-policy-agent/apiextensions/magalix.com/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type PoliciesInformer struct {
	store      cache.Store
	controller cache.Controller
	stop       chan struct{}
}

// NewPoliciesInformer returns PoliciesInformer to watch policies CRD
func NewPoliciesInformer(client *KubePoliciesClient, resoureceHandler cache.ResourceEventHandler, period time.Duration) *PoliciesInformer {
	listWatcher := cache.ListWatch{
		ListFunc: func(lo metav1.ListOptions) (result runtime.Object, err error) {
			return client.List(context.Background(), lo)
		},
		WatchFunc: func(lo metav1.ListOptions) (watch.Interface, error) {
			return client.Watch(context.Background(), lo)
		},
	}

	store, controller := cache.NewInformer(&listWatcher, &magalixv1.Policy{}, period, resoureceHandler)
	return &PoliciesInformer{
		store:      store,
		controller: controller,
		stop:       make(chan struct{}),
	}
}

// Start starts watching policies CRD wiat until a cache is built with current CRDs in the cluster
func (in *PoliciesInformer) Start() error {
	go in.controller.Run(in.stop)
	err := in.waitForCache()
	if err != nil {
		return err
	}
	return nil
}

// Stop stops the watcher
func (in *PoliciesInformer) Stop() {
	in.stop <- struct{}{}
}

func (in *PoliciesInformer) waitForCache() error {
	if !cache.WaitForCacheSync(in.stop, in.controller.HasSynced) {
		return fmt.Errorf("failed to build policies informer cache")
	}
	return nil
}

// List returns all the policies from the informer cache
func (in *PoliciesInformer) List() []*magalixv1.Policy {
	var policies []*magalixv1.Policy
	listResponse := in.store.List()

	for _, r := range listResponse {
		policies = append(policies, r.(*magalixv1.Policy))
	}
	return policies
}
