package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/http/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)


func TestClient_GetDataplane(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult *Dataplane
	}{
		{
			name:           "successful get",
			id:             "dp-123",
			responseStatus: 200,
			responseBody:   `{"id":"dp-123","name":"test-dataplane","type":"postgres","status":"active","config":{"host":"localhost"}}`,
			expectedResult: &Dataplane{
				ID:     "dp-123",
				Name:   "test-dataplane",
				Type:   "postgres",
				Status: "active",
				Config: map[string]string{"host": "localhost"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)
			mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
				StatusCode: tt.responseStatus,
				Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
			}, nil).Do(func(req *http.Request) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "/api/v1/dataplanes/"+tt.id, req.URL.Path)
			})

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

func TestClient_ListDataplanes(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult []Dataplane
	}{
		{
			name:           "successful list",
			responseStatus: 200,
			responseBody:   `[{"id":"dp-1","name":"dataplane-1","type":"postgres","status":"active","config":{}},{"id":"dp-2","name":"dataplane-2","type":"mysql","status":"inactive","config":{}}]`,
			expectedResult: []Dataplane{
				{ID: "dp-1", Name: "dataplane-1", Type: "postgres", Status: "active", Config: map[string]string{}},
				{ID: "dp-2", Name: "dataplane-2", Type: "mysql", Status: "inactive", Config: map[string]string{}},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)
			mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
				StatusCode: tt.responseStatus,
				Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
			}, nil).Do(func(req *http.Request) {
				assert.Equal(t, "GET", req.Method)
				assert.Equal(t, "/api/v1/dataplanes", req.URL.Path)
			})

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

func TestClient_CreateDataplane(t *testing.T) {
	tests := []struct {
		name           string
		spec           DataplaneSpec
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult *Dataplane
	}{
		{
			name: "successful create",
			spec: DataplaneSpec{
				Name:   "new-dataplane",
				Type:   "postgres",
				Config: map[string]string{"host": "localhost"},
			},
			responseStatus: 201,
			responseBody:   `{"id":"dp-new","name":"new-dataplane","type":"postgres","status":"creating","config":{"host":"localhost"}}`,
			expectedResult: &Dataplane{
				ID:     "dp-new",
				Name:   "new-dataplane",
				Type:   "postgres",
				Status: "creating",
				Config: map[string]string{"host": "localhost"},
			},
		},
		{
			name:           "validation error",
			spec:           DataplaneSpec{Name: ""},
			responseStatus: 400,
			responseBody:   `{"error":"Name is required"}`,
			expectedError:  "API error 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)
			mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
				StatusCode: tt.responseStatus,
				Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
			}, nil).Do(func(req *http.Request) {
				assert.Equal(t, "POST", req.Method)
				assert.Equal(t, "/api/v1/dataplanes", req.URL.Path)
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

				// Verify request body
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				var sentSpec DataplaneSpec
				err = json.Unmarshal(body, &sentSpec)
				require.NoError(t, err)
				assert.Equal(t, tt.spec, sentSpec)
			})

			client := NewClient(mockDoer)
			result, err := client.CreateDataplane(context.Background(), tt.spec)

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

func TestClient_DeleteDataplane(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		responseStatus int
		responseBody   string
		expectedError  string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDoer := mock.NewMockHTTPDoer(ctrl)
			mockDoer.EXPECT().Do(gomock.Any()).Return(&http.Response{
				StatusCode: tt.responseStatus,
				Body:       io.NopCloser(strings.NewReader(tt.responseBody)),
			}, nil).Do(func(req *http.Request) {
				assert.Equal(t, "DELETE", req.Method)
				assert.Equal(t, "/api/v1/dataplanes/"+tt.id, req.URL.Path)
			})

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

func TestClient_HTTPError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDoer := mock.NewMockHTTPDoer(ctrl)
	mockDoer.EXPECT().Do(gomock.Any()).Return(nil, assert.AnError)

	client := NewClient(mockDoer)

	_, err := client.GetDataplane(context.Background(), "dp-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}
