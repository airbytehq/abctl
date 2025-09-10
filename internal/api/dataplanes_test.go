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

func TestGetDataplane(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult *Dataplane
		httpError      error
	}{
		{
			name:           "successful get",
			id:             "dp-123",
			responseStatus: 200,
			responseBody:   `{"dataplaneId":"dp-123","name":"test-dataplane","regionId":"550e8400-e29b-41d4-a716-446655440000","enabled":true}`,
			expectedResult: &Dataplane{
				DataplaneID: "dp-123",
				Name:        "test-dataplane",
				RegionID:    "550e8400-e29b-41d4-a716-446655440000",
				Enabled:     true,
			},
		},
		{
			name:           "dataplane not found",
			id:             "nonexistent",
			responseStatus: 404,
			responseBody:   `{"error":"Dataplane not found"}`,
			expectedError:  "API error 404",
		},
		{
			name:           "invalid JSON response",
			id:             "dp-123",
			responseStatus: 200,
			responseBody:   `invalid json`,
			expectedError:  "failed to decode response",
		},
		{
			name:          "http client error",
			id:            "dp-123",
			httpError:     errors.New("network error"),
			expectedError: "request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)

			if tt.httpError != nil {
				mockDoer.EXPECT().Do(gomock.Any()).Return(nil, tt.httpError)
			} else {
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
				}, nil).Do(func(req *http.Request) {
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "/v1/dataplanes/"+tt.id, req.URL.Path)
				})
			}

			client := NewClient(mockDoer)
			result, err := client.GetDataplane(context.Background(), tt.id)

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

func TestListDataplanes(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult []Dataplane
		httpError      error
	}{
		{
			name:           "successful list",
			responseStatus: 200,
			responseBody:   `[{"dataplaneId":"dp-1","name":"dataplane-1","regionId":"550e8400-e29b-41d4-a716-446655440000","enabled":true},{"dataplaneId":"dp-2","name":"dataplane-2","regionId":"6ba7b810-9dad-11d1-80b4-00c04fd430c8","enabled":false}]`,
			expectedResult: []Dataplane{
				{DataplaneID: "dp-1", Name: "dataplane-1", RegionID: "550e8400-e29b-41d4-a716-446655440000", Enabled: true},
				{DataplaneID: "dp-2", Name: "dataplane-2", RegionID: "6ba7b810-9dad-11d1-80b4-00c04fd430c8", Enabled: false},
			},
		},
		{
			name:           "empty list",
			responseStatus: 200,
			responseBody:   `[]`,
			expectedResult: []Dataplane{},
		},
		{
			name:           "server error",
			responseStatus: 500,
			responseBody:   `{"error":"Internal server error"}`,
			expectedError:  "API error 500",
		},
		{
			name:          "http client error",
			httpError:     errors.New("network error"),
			expectedError: "request failed",
		},
		{
			name:           "json decode error",
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

			if tt.httpError != nil {
				mockDoer.EXPECT().Do(gomock.Any()).Return(nil, tt.httpError)
			} else {
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
				}, nil).Do(func(req *http.Request) {
					assert.Equal(t, "GET", req.Method)
					assert.Equal(t, "/v1/dataplanes", req.URL.Path)
				})
			}

			client := NewClient(mockDoer)
			result, err := client.ListDataplanes(context.Background())

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

func TestCreateDataplane(t *testing.T) {
	tests := []struct {
		name           string
		req            CreateDataplaneRequest
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult *CreateDataplaneResponse
		httpError      error
	}{
		{
			name: "successful create",
			req: CreateDataplaneRequest{
				Name:     "new-dataplane",
				RegionID: "550e8400-e29b-41d4-a716-446655440000",
				Enabled:  true,
			},
			responseStatus: 201,
			responseBody:   `{"dataplaneId":"dp-new","regionId":"550e8400-e29b-41d4-a716-446655440000","clientId":"client-123","clientSecret":"secret-456"}`,
			expectedResult: &CreateDataplaneResponse{
				DataplaneID:  "dp-new",
				RegionID:     "550e8400-e29b-41d4-a716-446655440000",
				ClientID:     "client-123",
				ClientSecret: "secret-456",
			},
		},
		{
			name:           "validation error",
			req:            CreateDataplaneRequest{Name: ""},
			responseStatus: 400,
			responseBody:   `{"error":"Name is required"}`,
			expectedError:  "API error 400",
		},
		{
			name:          "http client error",
			req:           CreateDataplaneRequest{Name: "test"},
			httpError:     errors.New("network error"),
			expectedError: "request failed",
		},
		{
			name:           "json decode error",
			req:            CreateDataplaneRequest{Name: "test"},
			responseStatus: 201,
			responseBody:   `invalid json`,
			expectedError:  "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)

			if tt.httpError != nil {
				mockDoer.EXPECT().Do(gomock.Any()).Return(nil, tt.httpError)
			} else {
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
				}, nil).Do(func(req *http.Request) {
					assert.Equal(t, "POST", req.Method)
					assert.Equal(t, "/v1/dataplanes", req.URL.Path)
					assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

					// Verify request body
					body, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					var sentReq CreateDataplaneRequest
					err = json.Unmarshal(body, &sentReq)
					require.NoError(t, err)
					assert.Equal(t, tt.req, sentReq)
				})
			}

			client := NewClient(mockDoer)
			result, err := client.CreateDataplane(context.Background(), tt.req)

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

func TestDeleteDataplane(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		responseStatus int
		responseBody   string
		expectedError  string
		httpError      error
	}{
		{
			name:           "successful delete - 204",
			id:             "dp-123",
			responseStatus: 204,
			responseBody:   "",
		},
		{
			name:           "successful delete - 200",
			id:             "dp-123",
			responseStatus: 200,
			responseBody:   `{"message":"Dataplane deleted"}`,
		},
		{
			name:           "dataplane not found",
			id:             "nonexistent",
			responseStatus: 404,
			responseBody:   `{"error":"Dataplane not found"}`,
			expectedError:  "API error 404",
		},
		{
			name:          "http client error",
			id:            "dp-123",
			httpError:     errors.New("network error"),
			expectedError: "request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)

			if tt.httpError != nil {
				mockDoer.EXPECT().Do(gomock.Any()).Return(nil, tt.httpError)
			} else {
				mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
					StatusCode: tt.responseStatus,
					Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
				}, nil).Do(func(req *http.Request) {
					assert.Equal(t, "DELETE", req.Method)
					assert.Equal(t, "/v1/dataplanes/"+tt.id, req.URL.Path)
				})
			}

			client := NewClient(mockDoer)
			err := client.DeleteDataplane(context.Background(), tt.id)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDoer := mock.NewMockHTTPDoer(ctrl)
	mockDoer.EXPECT().Do(gomock.Any()).Return(nil, assert.AnError)

	client := NewClient(mockDoer)

	_, err := client.GetDataplane(context.Background(), "dp-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}
