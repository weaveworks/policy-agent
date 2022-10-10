package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFluxObject(t *testing.T) {
	fluxObj := GetFluxObject(map[string]string{})

	if fluxObj != nil {
		t.Error("unexpected flux object")
	}

	for apiVersion, kind := range fluxControllerKindMap {
		fluxObj := GetFluxObject(map[string]string{
			fmt.Sprintf("%s/name", apiVersion):      "my-app",
			fmt.Sprintf("%s/namespace", apiVersion): "default",
		})

		assert.NotEqual(t, fluxObj, nil)

		assert.Equal(t, fluxObj.GetAPIVersion(), apiVersion)
		assert.Equal(t, fluxObj.GetKind(), kind)
		assert.Equal(t, fluxObj.GetNamespace(), "default")
		assert.Equal(t, fluxObj.GetName(), "my-app")
	}
}
