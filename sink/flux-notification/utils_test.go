package flux_notification

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGetFluxObject(t *testing.T) {
	fluxObj := getFluxObject(map[string]string{})
	assert.Equal(t, fluxObj, nil)

	for apiVersion, kind := range fluxControllerKindMap {
		fluxObj := getFluxObject(map[string]string{
			fmt.Sprintf("%s/name", apiVersion):      "my-app",
			fmt.Sprintf("%s/namespace", apiVersion): "default",
		})

		assert.NotEqual(t, fluxObj, nil)

		obj := fluxObj.(*unstructured.Unstructured)

		assert.Equal(t, obj.GetAPIVersion(), apiVersion)
		assert.Equal(t, obj.GetKind(), kind)
		assert.Equal(t, obj.GetNamespace(), "default")
		assert.Equal(t, obj.GetName(), "my-app")
	}
}
