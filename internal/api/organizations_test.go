package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/http/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestGetOrganization(t *testing.T) {
	tests := []struct {
		name           string
		organizationID string
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult *Organization
		httpError      error
	}{
		{
			name:           "successful get",
			organizationID: "550e8400-e29b-41d4-a716-446655440000",
			responseStatus: 200,
			responseBody:   `{"organizationId":"550e8400-e29b-41d4-a716-446655440000","organizationName":"Test Organization","email":"test@example.com"}`,
			expectedResult: &Organization{
				ID:    "550e8400-e29b-41d4-a716-446655440000",
				Name:  "Test Organization",
				Email: "test@example.com",
			},
		},
		{
			name:           "organization not found",
			organizationID: "invalid-id",
			responseStatus: 404,
			responseBody:   `{"error":"Organization not found"}`,
			expectedError:  "API error 404",
		},
		{
			name:           "http client error",
			organizationID: "test-id",
			httpError:      errors.New("network error"),
			expectedError:  "request failed",
		},
		{
			name:           "json decode error",
			organizationID: "test-id",
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
					assert.Equal(t, "/v1/organizations/"+tt.organizationID, req.URL.Path)
				})
			}

			client := NewClient(mockDoer)
			result, err := client.GetOrganization(context.Background(), tt.organizationID)

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

func TestListOrganizations(t *testing.T) {
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  string
		expectedResult []*Organization
		httpError      error
	}{
		{
			name:           "success",
			responseStatus: 200,
			responseBody:   `{"data":[{"organizationId":"550e8400-e29b-41d4-a716-446655440000","organizationName":"Test Organization","email":"test@example.com"},{"organizationId":"6ba7b810-9dad-11d1-80b4-00c04fd430c8","organizationName":"Another Organization","email":"another@example.com"}]}`,
			expectedResult: []*Organization{
				{
					ID:    "550e8400-e29b-41d4-a716-446655440000",
					Name:  "Test Organization",
					Email: "test@example.com",
				},
				{
					ID:    "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
					Name:  "Another Organization",
					Email: "another@example.com",
				},
			},
		},
		{
			name:           "empty data",
			responseStatus: 200,
			responseBody:   `{"data":[]}`,
			expectedResult: []*Organization{},
		},
		{
			name:           "API error",
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
					assert.Equal(t, "/v1/organizations", req.URL.Path)
				})
			}

			client := NewClient(mockDoer)
			result, err := client.ListOrganizations(context.Background())

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

func TestWithNameContains(t *testing.T) {
	tests := []struct {
		name         string
		searchName   string
		expectedName string
	}{
		{
			name:         "sets name filter",
			searchName:   "test-org",
			expectedName: "test-org",
		},
		{
			name:         "sets empty name",
			searchName:   "",
			expectedName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ListOrganizationsOptions{}
			option := WithNameContains(tt.searchName)
			option(opts)
			assert.Equal(t, tt.expectedName, opts.NameContains)
		})
	}
}

func TestWithPagination(t *testing.T) {
	tests := []struct {
		name              string
		pagination        *Pagination
		expectedPageSize  int
		expectedRowOffset int
	}{
		{
			name: "sets pagination",
			pagination: &Pagination{
				PageSize:  50,
				RowOffset: 100,
			},
			expectedPageSize:  50,
			expectedRowOffset: 100,
		},
		{
			name: "sets default pagination",
			pagination: &Pagination{
				PageSize:  20,
				RowOffset: 0,
			},
			expectedPageSize:  20,
			expectedRowOffset: 0,
		},
		{
			name:              "nil pagination",
			pagination:        nil,
			expectedPageSize:  0,
			expectedRowOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &ListOrganizationsOptions{}
			option := WithPagination(tt.pagination)
			option(opts)

			if tt.pagination != nil {
				assert.NotNil(t, opts.Pagination)
				assert.Equal(t, tt.expectedPageSize, opts.Pagination.PageSize)
				assert.Equal(t, tt.expectedRowOffset, opts.Pagination.RowOffset)
			} else {
				assert.Nil(t, opts.Pagination)
			}
		})
	}
}
