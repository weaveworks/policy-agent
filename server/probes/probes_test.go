package probes

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewProbesHandler(t *testing.T) {
	ready := &atomic.Value{}
	ready.Store(false)
	tests := []struct {
		name    string
		address string
		want    *ProbesHandler
	}{
		{
			name:    "default test",
			address: ":8000",
			want: &ProbesHandler{
				address: ":8000",
				ready:   ready,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewProbesHandler(tt.address); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProbesHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProbesHandler_healthHandler(t *testing.T) {
	type fields struct {
		address string
		ready   *atomic.Value
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "default test",
			fields: fields{
				address: ":8000",
				ready:   &atomic.Value{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := require.New(t)
			p := &ProbesHandler{
				address: tt.fields.address,
				ready:   tt.fields.ready,
			}
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()
			p.healthHandler(w, req)
			res := w.Result()
			assert.Equal(http.StatusOK, res.StatusCode)
		})
	}
}

func TestProbesHandler_readinessHandler(t *testing.T) {
	type fields struct {
		address string
		ready   *atomic.Value
	}
	tests := []struct {
		name           string
		fields         fields
		readyVal       bool
		wantStatusCode int
	}{
		{
			name: "default test",
			fields: fields{
				address: ":8000",
				ready:   &atomic.Value{},
			},
			readyVal:       true,
			wantStatusCode: http.StatusOK,
		},
		{
			name: "not ready test",
			fields: fields{
				address: ":8000",
				ready:   &atomic.Value{},
			},
			readyVal:       false,
			wantStatusCode: http.StatusServiceUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := require.New(t)
			p := &ProbesHandler{
				address: tt.fields.address,
				ready:   tt.fields.ready,
			}

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			w := httptest.NewRecorder()
			p.MarkReady(tt.readyVal)
			p.readinessHandler(w, req)
			res := w.Result()
			assert.Equal(tt.wantStatusCode, res.StatusCode)
		})
	}
}
