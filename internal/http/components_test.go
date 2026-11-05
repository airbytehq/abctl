package http

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type componentRouteDoer func(*http.Request) (*http.Response, error)

func (f componentRouteDoer) Do(req *http.Request) (*http.Response, error) { return f(req) }

func TestConfigAPIRouting(t *testing.T) {
	for _, tc := range []struct{ path, expected string }{
		{"/v1/dataplanes", "https://api.example.com/v1/dataplanes"},
		{"/v1/applications/token", "https://api.example.com/v1/applications/token"},
		{"/api/v1/dataplanes/register_component", "https://cloud.example.com/api/v1/dataplanes/register_component"},
		{"/api/v1/dataplanes/unregister_component", "https://cloud.example.com/api/v1/dataplanes/unregister_component"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client, err := NewClient("https://api.example.com", componentRouteDoer(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, tc.expected, req.URL.String())
				require.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
				require.Equal(t, ctx, req.Context())
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("{}"))}, nil
			}))
			require.NoError(t, err)
			client.WithConfigAPI("https://cloud.example.com")
			req, err := http.NewRequestWithContext(ctx, "POST", tc.path, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer test-token")
			resp, err := client.Do(req)
			require.NoError(t, err)
			resp.Body.Close()
		})
	}
}
