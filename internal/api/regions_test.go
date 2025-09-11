package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/http/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetRegion(t *testing.T) {
	tests := []struct {
		name           string
		regionID       string
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult *Region
		httpError      error
		setupMock      func(*mock.MockHTTPDoer)
	}{
		{
			name:           "successful get",
			regionID:       "550e8400-e29b-41d4-a716-446655440000",
			responseStatus: 200,
			responseBody:   `{"regionId":"550e8400-e29b-41d4-a716-446655440000","name":"us-east-1","cloudProvider":"AWS","location":"US East","status":"active"}`,
			expectedResult: &Region{
				ID:            "550e8400-e29b-41d4-a716-446655440000",
				Name:          "us-east-1",
				CloudProvider: "AWS",
				Location:      "US East",
				Status:        "active",
			},
		},
		{
			name:           "region not found",
			regionID:       "invalid-id",
			responseStatus: 404,
			responseBody:   `{"error":"Region not found"}`,
			expectedError:  "API error 404",
		},
		{
			name:          "http client error",
			regionID:      "test-id",
			httpError:     errors.New("network error"),
			expectedError: "request failed",
		},
		{
			name:           "json decode error",
			regionID:       "test-id",
			responseStatus: 200,
			responseBody:   `invalid json`,
			expectedError:  "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(mockDoer)
			} else if tt.httpError != nil {
				mockDoer.EXPECT().Do(gomock.Any()).Return(nil, tt.httpError)
			} else {
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
				}, nil).Do(func(req *http.Request) {
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "/v1/regions/"+tt.regionID, req.URL.Path)
				})
			}

			client := NewClient(mockDoer)
			result, err := client.GetRegion(context.Background(), tt.regionID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestListRegions(t *testing.T) {
	tests := []struct {
		name           string
		organizationID string
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult []*Region
		httpError      error
		setupMock      func(*mock.MockHTTPDoer)
	}{
		{
			name:           "successful list",
			organizationID: "org-123",
			responseStatus: 200,
			responseBody:   `[{"regionId":"550e8400-e29b-41d4-a716-446655440000","name":"us-east-1","cloudProvider":"AWS","location":"US East","status":"active"},{"regionId":"6ba7b810-9dad-11d1-80b4-00c04fd430c8","name":"eu-west-1","cloudProvider":"AWS","location":"EU West","status":"active"}]`,
			expectedResult: []*Region{
				{
					ID:            "550e8400-e29b-41d4-a716-446655440000",
					Name:          "us-east-1",
					CloudProvider: "AWS",
					Location:      "US East",
					Status:        "active",
				},
				{
					ID:            "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					Name:          "eu-west-1",
					CloudProvider: "AWS",
					Location:      "EU West",
					Status:        "active",
				},
			},
		},
		{
			name:           "API error",
			organizationID: "org-123",
			responseStatus: 500,
			responseBody:   `{"error":"Internal server error"}`,
			expectedError:  "API error 500",
		},
		{
			name:           "http client error",
			organizationID: "org-123",
			httpError:      errors.New("network error"),
			expectedError:  "request failed",
		},
		{
			name:           "json decode error",
			organizationID: "org-123",
			responseStatus: 200,
			responseBody:   `invalid json`,
			expectedError:  "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(mockDoer)
			} else if tt.httpError != nil {
				mockDoer.EXPECT().Do(gomock.Any()).Return(nil, tt.httpError)
			} else {
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
				}, nil).Do(func(req *http.Request) {
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "/v1/regions", req.URL.Path)
					if tt.organizationID != "" {
						assert.Equal(t, tt.organizationID, req.URL.Query().Get("organizationId"))
					}
				})
			}

			client := NewClient(mockDoer)
			result, err := client.ListRegions(context.Background(), tt.organizationID)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestCreateRegion(t *testing.T) {
	tests := []struct {
		name           string
		request        CreateRegionRequest
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult *Region
		httpError      error
		setupMock      func(*mock.MockHTTPDoer)
	}{
		{
			name: "successful create",
			request: CreateRegionRequest{
				Name:          "us-west-2",
				CloudProvider: "AWS",
				Location:      "US West",
			},
			responseStatus: 201,
			responseBody:   `{"regionId":"650e8400-e29b-41d4-a716-446655440000","name":"us-west-2","cloudProvider":"AWS","location":"US West","status":"active"}`,
			expectedResult: &Region{
				ID:            "650e8400-e29b-41d4-a716-446655440000",
				Name:          "us-west-2",
				CloudProvider: "AWS",
				Location:      "US West",
				Status:        "active",
			},
		},
		{
			name: "validation error",
			request: CreateRegionRequest{
				Name: "", // Invalid - empty name
			},
			responseStatus: 400,
			responseBody:   `{"error":"Name is required"}`,
			expectedError:  "API error 400",
		},
		{
			name: "conflict error",
			request: CreateRegionRequest{
				Name:          "us-east-1",
				CloudProvider: "AWS",
				Location:      "US East",
			},
			responseStatus: 409,
			responseBody:   `{"error":"Region already exists"}`,
			expectedError:  "API error 409",
		},
		{
			name: "json decode error",
			request: CreateRegionRequest{
				Name: "test-region",
			},
			responseStatus: 201,
			responseBody:   `invalid json`,
			expectedError:  "failed to decode response",
		},
		{
			name: "read response body error",
			request: CreateRegionRequest{
				Name: "test-region",
			},
			expectedError: "failed to read response body",
			setupMock: func(mockDoer *mock.MockHTTPDoer) {
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: 201,
					Body:       &errorReader{},
				}, nil)
			},
		},
		{
			name: "http client error",
			request: CreateRegionRequest{
				Name: "test-region",
			},
			httpError:     errors.New("network error"),
			expectedError: "request failed",
		},
		{
			name: "server error",
			request: CreateRegionRequest{
				Name: "test-region",
			},
			responseStatus: 500,
			responseBody:   `{"error":"Internal server error"}`,
			expectedError:  "API error 500",
		},
		{
			name: "unauthorized error",
			request: CreateRegionRequest{
				Name: "test-region",
			},
			responseStatus: 401,
			responseBody:   `{"error":"Unauthorized"}`,
			expectedError:  "API error 401",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)

			if tt.setupMock != nil {
				tt.setupMock(mockDoer)
			} else if tt.httpError != nil {
				mockDoer.EXPECT().Do(gomock.Any()).Return(nil, tt.httpError)
			} else {
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
				}, nil).Do(func(req *http.Request) {
					assert.Equal(t, "POST", req.Method)
					assert.Equal(t, "/v1/regions", req.URL.Path)
					assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

					// Verify request body
					var sentRequest CreateRegionRequest
					err := json.NewDecoder(req.Body).Decode(&sentRequest)
					require.NoError(t, err)
					assert.Equal(t, tt.request, sentRequest)
				})
			}

			client := NewClient(mockDoer)
			result, err := client.CreateRegion(context.Background(), tt.request)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
