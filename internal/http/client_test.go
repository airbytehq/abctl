package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/http/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testKey string

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{
			name:    "success",
			baseURL: "https://api.example.com",
		},
		{
			name:    "empty base URL",
			baseURL: "",
		},
		{
			name:    "URL with path",
			baseURL: "https://api.example.com/v1",
		},
		{
			name:    "nil doer",
			baseURL: "https://api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var doer HTTPDoer
			if tt.name != "nil doer" {
				doer = mock.NewMockHTTPDoer(ctrl)
			}

			client, err := NewClient(tt.baseURL, doer)

			assert.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tt.baseURL, client.baseURL)
			if tt.name == "nil doer" {
				assert.Nil(t, client.doer)
			} else {
				assert.Equal(t, doer, client.doer)
			}
		})
	}
}

func TestClient_Do(t *testing.T) {
	tests := []struct {
		name          string
		baseURL       string
		requestPath   string
		method        string
		body          string
		headers       map[string]string
		expectURL     string
		setupMocks    func(ctrl *gomock.Controller) HTTPDoer
		expectedError string
	}{
		{
			name:        "success",
			baseURL:     "https://api.example.com",
			requestPath: "/api/v1/dataplanes",
			method:      "GET",
			expectURL:   "https://api.example.com/api/v1/dataplanes",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				mockDoer := mock.NewMockHTTPDoer(ctrl)
				mockDoer.EXPECT().Do(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "https://api.example.com/api/v1/dataplanes", req.URL.String())
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "test-value", req.Context().Value(testKey("test-key")))
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("response body")),
					}, nil
				})
				return mockDoer
			},
		},
		{
			name:        "base URL with path",
			baseURL:     "https://api.example.com/control",
			requestPath: "/api/v1/dataplanes",
			method:      "GET",
			expectURL:   "https://api.example.com/control/api/v1/dataplanes",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				mockDoer := mock.NewMockHTTPDoer(ctrl)
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("response body")),
				}, nil)
				return mockDoer
			},
		},
		{
			name:        "query params",
			baseURL:     "https://api.example.com",
			requestPath: "/api/v1/dataplanes?limit=10",
			method:      "GET",
			expectURL:   "https://api.example.com/api/v1/dataplanes?limit=10",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				mockDoer := mock.NewMockHTTPDoer(ctrl)
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("response body")),
				}, nil)
				return mockDoer
			},
		},
		{
			name:        "POST with body",
			baseURL:     "https://api.example.com",
			requestPath: "/api/v1/dataplanes",
			method:      "POST",
			body:        `{"name":"test-dataplane"}`,
			headers:     map[string]string{"Content-Type": "application/json"},
			expectURL:   "https://api.example.com/api/v1/dataplanes",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				mockDoer := mock.NewMockHTTPDoer(ctrl)
				mockDoer.EXPECT().Do(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					body, _ := io.ReadAll(req.Body)
					assert.Equal(t, `{"name":"test-dataplane"}`, string(body))
					assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("response body")),
					}, nil
				})
				return mockDoer
			},
		},
		{
			name:        "PUT with headers",
			baseURL:     "https://api.example.com",
			requestPath: "/api/v1/dataplanes/123",
			method:      "PUT",
			body:        `{"name":"updated-dataplane"}`,
			headers:     map[string]string{"Content-Type": "application/json", "Authorization": "Bearer token"},
			expectURL:   "https://api.example.com/api/v1/dataplanes/123",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				mockDoer := mock.NewMockHTTPDoer(ctrl)
				mockDoer.EXPECT().Do(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
					assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
					assert.Equal(t, "Bearer token", req.Header.Get("Authorization"))
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("response body")),
					}, nil
				})
				return mockDoer
			},
		},
		{
			name:        "DELETE request",
			baseURL:     "https://api.example.com",
			requestPath: "/api/v1/dataplanes/123",
			method:      "DELETE",
			expectURL:   "https://api.example.com/api/v1/dataplanes/123",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				mockDoer := mock.NewMockHTTPDoer(ctrl)
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("response body")),
				}, nil)
				return mockDoer
			},
		},
		{
			name:        "root path",
			baseURL:     "https://api.example.com",
			requestPath: "/",
			method:      "GET",
			expectURL:   "https://api.example.com/",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				mockDoer := mock.NewMockHTTPDoer(ctrl)
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("response body")),
				}, nil)
				return mockDoer
			},
		},
		{
			name:        "empty path",
			baseURL:     "https://api.example.com",
			requestPath: "",
			method:      "GET",
			expectURL:   "https://api.example.com",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				mockDoer := mock.NewMockHTTPDoer(ctrl)
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("response body")),
				}, nil)
				return mockDoer
			},
		},
		{
			name:          "doer error",
			baseURL:       "https://api.example.com",
			requestPath:   "/test/path",
			method:        "GET",
			expectURL:     "https://api.example.com/test/path",
			expectedError: "mock error",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				mockDoer := mock.NewMockHTTPDoer(ctrl)
				mockDoer.EXPECT().Do(gomock.Any()).Return(nil, errors.New("mock error"))
				return mockDoer
			},
		},
		{
			name:          "nil doer error",
			baseURL:       "https://api.example.com",
			requestPath:   "/test",
			method:        "GET",
			expectedError: "nil pointer",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				return nil
			},
		},
		{
			name:          "invalid base URL",
			baseURL:       ":",
			requestPath:   "/test",
			method:        "GET",
			expectedError: "failed to join API base URL with request path",
			setupMocks: func(ctrl *gomock.Controller) HTTPDoer {
				return mock.NewMockHTTPDoer(ctrl)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			doer := tt.setupMocks(ctrl)
			client, err := NewClient(tt.baseURL, doer)
			require.NoError(t, err)

			reqURL, err := url.Parse(tt.requestPath)
			require.NoError(t, err)

			req := &http.Request{
				Method: tt.method,
				URL:    reqURL,
				Header: make(http.Header),
			}

			if tt.body != "" {
				req.Body = io.NopCloser(strings.NewReader(tt.body))
			}

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ctx := context.WithValue(context.Background(), testKey("test-key"), "test-value")
			req = req.WithContext(ctx)

			resp, err := client.Do(req)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()
		})
	}
}
