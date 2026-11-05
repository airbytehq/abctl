package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type componentDoer func(*http.Request) (*http.Response, error)

func (f componentDoer) Do(req *http.Request) (*http.Response, error) { return f(req) }

func TestComponentAPI(t *testing.T) {
	for _, tc := range []struct {
		name      string
		body      string
		status    int
		wantError string
	}{
		{"credentials", `{"clientId":"install-client","clientSecret":"unit-test-only"}`, 200, ""},
		{"URL only", `{}`, 200, ""},
		{"malformed JSON", `{"clientSecret":"never-echo-this"`, 200, "not valid JSON"},
		{"rejected", `{"clientSecret":"never-echo-this"}`, 422, "HTTP 422"},
		{"wrong success status", `{}`, 204, "HTTP 204"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClient(componentDoer(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, "POST", req.Method)
				require.Equal(t, "/api/v1/dataplanes/register_component", req.URL.Path)
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				var body RegisterComponentRequest
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				require.Equal(t, RegisterComponentRequest{"dp", "cloud_mcp", "https://mcp.example.com"}, body)
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			}))
			credentials, err := client.RegisterComponent(context.Background(), RegisterComponentRequest{"dp", "cloud_mcp", "https://mcp.example.com"})
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				require.NotContains(t, err.Error(), "never-echo-this")
				require.Nil(t, credentials)
			} else {
				require.NoError(t, err)
				if tc.name == "credentials" {
					require.Equal(t, "install-client", credentials.ClientID)
					require.Equal(t, "unit-test-only", credentials.ClientSecret)
				} else {
					require.Empty(t, credentials.ClientSecret)
				}
			}
		})
	}
	client := NewClient(componentDoer(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "/api/v1/dataplanes/unregister_component", req.URL.Path)
		var body map[string]string
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, map[string]string{"dataplaneId": "dp", "component": "agui"}, body)
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader(""))}, nil
	}))
	for i := 0; i < 2; i++ {
		require.NoError(t, client.UnregisterComponent(context.Background(), UnregisterComponentRequest{"dp", "agui"}))
	}
	client = NewClient(componentDoer(func(*http.Request) (*http.Response, error) { return nil, errors.New("network unavailable") }))
	_, err := client.RegisterComponent(context.Background(), RegisterComponentRequest{})
	require.ErrorContains(t, err, "network unavailable")
	require.ErrorContains(t, client.UnregisterComponent(context.Background(), UnregisterComponentRequest{}), "network unavailable")
}
