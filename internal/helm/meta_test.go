package helm

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/airbytehq/abctl/internal/helm/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"helm.sh/helm/v3/pkg/chart"
)

func TestGetMetadataForRef(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockClient(ctrl)

	tests := []struct {
		name      string
		chartRef  string
		setupMock func()
		wantMeta  *chart.Metadata
		wantErr   bool
	}{
		{
			name:     "success",
			chartRef: "valid",
			setupMock: func() {
				mockClient.EXPECT().
					GetChart("valid", gomock.Any()).
					Return(&chart.Chart{Metadata: &chart.Metadata{Version: "1.2.3"}}, "", nil)
			},
			wantMeta: &chart.Metadata{Version: "1.2.3"},
		},
		{
			name:     "error from client",
			chartRef: "bad",
			setupMock: func() {
				mockClient.EXPECT().
					GetChart("bad", gomock.Any()).
					Return(nil, "", errors.New("fail"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}
			meta, err := GetMetadataForRef(mockClient, tt.chartRef)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMeta, meta)
			}
		})
	}
}

func TestGetMetadataForURL(t *testing.T) {
	t.Parallel()

	// Serve a valid chart archive with Chart.yaml for testing URL-based chart resolution
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "notfound") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte("not a real tgz"))
	}))
	defer ts.Close()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid but invalid archive", ts.URL + "/chart.tgz", true},
		{"404", ts.URL + "/notfound", true},
		{"unreachable", "http://127.0.0.1:0/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetMetadataForURL(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
