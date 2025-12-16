package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	httpmock "github.com/airbytehq/abctl/internal/http/mock"
)

func TestCredentialsFromJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		expectErr error
	}{
		{
			name:     "valid json",
			jsonData: `{"access_token":"test-token","refresh_token":"test-refresh","token_type":"Bearer","expires_at":"2023-01-01T12:00:00Z"}`,
		},
		{
			name:      "invalid json",
			jsonData:  `{"invalid": json}`,
			expectErr: errors.New("invalid character 'j' looking for beginning of value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds, err := credentialsFromJSON([]byte(tt.jsonData))

			if tt.expectErr == nil {
				require.NoError(t, err)
				return
			}
			assert.Equal(t, tt.expectErr.Error(), err.Error())
			assert.Nil(t, creds)
		})
	}
}

func TestDiscoverProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTPClient := httpmock.NewMockHTTPDoer(ctrl)

	// Mock successful OIDC discovery response
	discoveryResponse := `{
		"issuer": "https://example.com",
		"authorization_endpoint": "https://example.com/auth",
		"token_endpoint": "https://example.com/token"
	}`

	mockHTTPClient.EXPECT().Do(gomock.Any()).Return(&http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(discoveryResponse)),
	}, nil)

	provider, err := DiscoverProvider(context.Background(), "https://example.com", mockHTTPClient)
	if err != nil {
		t.Fatalf("DiscoverProvider failed: %v", err)
	}
	if provider == nil {
		t.Fatal("expected provider, got nil")
	}
}

func TestDiscoverProviderWithClient(t *testing.T) {
	tests := []struct {
		name      string
		setupHTTP func(mock *httpmock.MockHTTPDoer)
		expectErr error
	}{
		{
			name: "success",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				response := `{
					"issuer": "https://example.com",
					"authorization_endpoint": "https://example.com/auth",
					"token_endpoint": "https://example.com/token"
				}`
				mock.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(response)),
				}, nil)
			},
		},
		{
			name: "http error",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				mock.EXPECT().Do(gomock.Any()).Return(nil, errors.New("network error"))
			},
			expectErr: errors.New("failed to fetch provider configuration: network error"),
		},
		{
			name: "non-200 status",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				mock.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(strings.NewReader("Not Found")),
				}, nil)
			},
			expectErr: errors.New("discovery failed with status 404"),
		},
		{
			name: "invalid json response",
			setupHTTP: func(mock *httpmock.MockHTTPDoer) {
				mock.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("invalid json")),
				}, nil)
			},
			expectErr: errors.New("failed to decode provider configuration: invalid character 'i' looking for beginning of value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockHTTPClient := httpmock.NewMockHTTPDoer(ctrl)
			tt.setupHTTP(mockHTTPClient)

			provider, err := DiscoverProvider(context.Background(), "https://example.com", mockHTTPClient)

			if tt.expectErr == nil {
				require.NoError(t, err)
				assert.NotNil(t, provider)
				return
			}
			assert.Equal(t, tt.expectErr.Error(), err.Error())
		})
	}
}
