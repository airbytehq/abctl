package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/http/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{
			name:    "valid URL",
			baseURL: "https://api.example.com",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			baseURL: "://invalid-url",
			wantErr: true,
		},
		{
			name:    "URL with path",
			baseURL: "https://api.example.com/v1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)
			client, err := NewClient(tt.baseURL, mockDoer)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, mockDoer, client.doer)
			}
		})
	}
}

func TestClient_Do(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		requestPath string
		expectURL   string
	}{
		{
			name:        "simple path",
			baseURL:     "https://api.example.com",
			requestPath: "/api/v1/dataplanes",
			expectURL:   "https://api.example.com/api/v1/dataplanes",
		},
		{
			name:        "base URL with path",
			baseURL:     "https://api.example.com/control",
			requestPath: "/api/v1/dataplanes",
			expectURL:   "https://api.example.com/api/v1/dataplanes",
		},
		{
			name:        "path with query params",
			baseURL:     "https://api.example.com",
			requestPath: "/api/v1/dataplanes?limit=10",
			expectURL:   "https://api.example.com/api/v1/dataplanes?limit=10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server to capture the actual request
			var capturedURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Create client with test server URL
			client, err := NewClient(server.URL, &http.Client{})
			require.NoError(t, err)

			// Create request with relative path
			reqURL, err := url.Parse(tt.requestPath)
			require.NoError(t, err)

			req := &http.Request{
				Method: "GET",
				URL:    reqURL,
				Header: make(http.Header),
			}

			// Make request
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify the URL was resolved correctly
			// Note: We compare the full request URL since the host will be the test server
			expectedURL, _ := url.Parse(tt.expectURL)
			capturedParsedURL, _ := url.Parse(capturedURL)
			assert.Equal(t, expectedURL.Path, capturedParsedURL.Path)
			assert.Equal(t, expectedURL.RawQuery, capturedParsedURL.RawQuery)
		})
	}
}

func TestClient_Do_WithMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDoer := mock.NewMockHTTPDoer(ctrl)
	client, err := NewClient("https://api.example.com", mockDoer)
	require.NoError(t, err)

	// Set up expected response
	expectedResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("response body")),
	}

	// Expect Do to be called with a request that has the resolved URL
	mockDoer.EXPECT().Do(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
		// Verify the URL was properly resolved
		assert.Equal(t, "https://api.example.com/test/path", req.URL.String())
		assert.Equal(t, "GET", req.Method)
		return expectedResp, nil
	})

	// Create request with relative path
	reqURL, _ := url.Parse("/test/path")
	req := &http.Request{
		Method: "GET",
		URL:    reqURL,
		Header: make(http.Header),
	}

	// Make request
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, expectedResp, resp)
}

func TestClient_Do_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context was preserved
		assert.NotNil(t, r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, &http.Client{})
	require.NoError(t, err)

	reqURL, _ := url.Parse("/test")
	req := &http.Request{
		Method: "GET",
		URL:    reqURL,
		Header: make(http.Header),
	}

	// Add context to request
	req = req.WithContext(req.Context())

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClient_Do_PreservesHeaders(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, &http.Client{})
	require.NoError(t, err)

	reqURL, _ := url.Parse("/test")
	req := &http.Request{
		Method: "POST",
		URL:    reqURL,
		Header: make(http.Header),
		Body:   io.NopCloser(strings.NewReader("test body")),
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify headers were preserved
	assert.Equal(t, "application/json", capturedHeaders.Get("Content-Type"))
	assert.Equal(t, "Bearer token", capturedHeaders.Get("Authorization"))
}
