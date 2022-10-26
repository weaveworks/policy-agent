package crd

import (
	"context"
	"fmt"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cacheMock struct {
	informertest.FakeInformers
	items map[reflect.Type]client.ObjectList
}

func (c *cacheMock) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	items, ok := c.items[reflect.TypeOf(list).Elem()]
	if !ok {
		return fmt.Errorf("invalid item type")
	}
	reflect.ValueOf(list).Elem().Set(reflect.ValueOf(items).Elem())
	return nil
}
