package helm

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airbytehq/abctl/internal/helm/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"helm.sh/helm/v3/pkg/chart"
)

func TestChartIsV2Plus(t *testing.T) {
	tests := []struct {
		name string
		ver  string
		want bool
	}{
		{
			name: "empty version",
			ver:  "",
			want: false,
		},
		{
			name: "v1 version",
			ver:  "1.0.0",
			want: false,
		},
		{
			name: "v1 with v prefix",
			ver:  "v1.0.0",
			want: false,
		},
		{
			name: "v2 version",
			ver:  "2.0.0",
			want: true,
		},
		{
			name: "v2 with v prefix",
			ver:  "v2.0.0",
			want: true,
		},
		{
			name: "v2 pre-release",
			ver:  "v2.0.0-alpha.1",
			want: false,
		},
		{
			name: "v3 version",
			ver:  "3.0.0",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChartIsV2Plus(tt.ver)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveChartReference(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock HTTP server for v1 Helm repo index
	v1Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `apiVersion: v1
entries:
  airbyte:
    - name: "airbyte"
      version: "1.6.0"
      urls: ["airbyte-1.6.0.tgz"]
    - name: "airbyte"
      version: "1.7.0"
      urls: ["airbyte-1.7.0.tgz"]
    - name: "airbyte"
      version: "1.8.0-alpha.1"
      urls: ["airbyte-1.8.0-alpha.1.tgz"]
`)
	}))
	defer v1Server.Close()

	// Mock HTTP server for v2 Helm repo index
	v2Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `apiVersion: v1
entries:
  airbyte:
    - name: "airbyte"
      version: "2.0.6"
      urls: ["airbyte-2.0.6.tgz"]
    - name: "airbyte"
      version: "2.0.0"
      urls: ["airbyte-2.0.0.tgz"]
    - name: "airbyte"
      version: "2.1.0-beta.1"
      urls: ["airbyte-2.1.0-beta.1.tgz"]
`)
	}))
	defer v2Server.Close()

	// Mock HTTP server for chart files
	chartData, _ := base64.StdEncoding.DecodeString("H4sIAAAAAAAA/ypJLS7RTc5ILCrRZ6AVMDAwMDA3NQXTBgYG6DQm29DQyMyYQcGUZi5CAqXFJYlFDAYGlJqD7rkhApDi3xlE6lUm5uZQ2Q5Q" +
		"eJhhxjsizk3N0eLf2MjAiEGBLoE4wuM/sSAzLLWoODM/z0qhzIgrLzE31UoBkSi4ymCShnoGegZcA+3cUTAKRsEoGAVUAoAAAAD//6flVyUA" +
		"CgAA")
	chartServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(chartData)
	}))
	defer chartServer.Close()

	// Mock HTTP server for v2 error case
	v2ServerWithError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer v2ServerWithError.Close()

	mockClient := mock.NewMockClient(ctrl)
	resolver := NewChartResolverWithURLs(mockClient, v1Server.URL, v2Server.URL)
	resolverWithError := NewChartResolverWithURLs(mockClient, v1Server.URL, v2ServerWithError.URL)

	tests := []struct {
		name     string
		chart    string
		version  string
		resolver *ChartResolver
		// setupMock allows test cases to configure mock client behavior for local chart and URL operations
		setupMock func()
		wantChart string
		wantVer   string
		wantErr   bool
	}{
		{
			name:      "empty chart and version gets latest v2",
			chart:     "",
			version:   "",
			wantChart: fmt.Sprintf("%s/airbyte-%s.tgz", v2Server.URL, "2.0.6"),
			wantVer:   "2.0.6",
		},
		{
			name:     "empty chart and version with v2 repo error",
			chart:    "",
			version:  "",
			resolver: resolverWithError,
			wantErr:  true,
		},
		{
			name:      "version only v1",
			chart:     "",
			version:   "1.6.0",
			wantChart: fmt.Sprintf("%s/airbyte-%s.tgz", v1Server.URL, "1.6.0"),
			wantVer:   "1.6.0",
		},
		{
			name:      "version only v2",
			chart:     "",
			version:   "2.0.0",
			wantChart: fmt.Sprintf("%s/airbyte-%s.tgz", v2Server.URL, "2.0.0"),
			wantVer:   "2.0.0",
		},
		{
			name:      "version only v1 pre-release",
			chart:     "",
			version:   "1.8.0-alpha.1",
			wantChart: fmt.Sprintf("%s/airbyte-%s.tgz", v1Server.URL, "1.8.0-alpha.1"),
			wantVer:   "1.8.0-alpha.1",
		},
		{
			name:      "version only v2 pre-release",
			chart:     "",
			version:   "2.1.0-beta.1",
			wantChart: fmt.Sprintf("%s/airbyte-%s.tgz", v2Server.URL, "2.1.0-beta.1"),
			wantVer:   "2.1.0-beta.1",
		},
		{
			name:      "chart as local ref",
			chart:     "local/path",
			version:   "",
			wantChart: "local/path",
			wantVer:   "2.0.0",
			setupMock: func() {
				mockClient.EXPECT().
					GetChart("local/path", gomock.Any()).
					Return(&chart.Chart{Metadata: &chart.Metadata{Version: "2.0.0"}}, "", nil)
			},
		},
		{
			name:      "chart as URL",
			chart:     fmt.Sprintf("%s/airbyte-1.0.0.tgz", chartServer.URL),
			version:   "",
			wantChart: fmt.Sprintf("%s/airbyte-1.0.0.tgz", chartServer.URL),
			wantVer:   "1.0.0",
		},
		{
			name:    "chart as URL with error",
			chart:   "https://invalid.example.com/chart.tgz",
			version: "",
			wantErr: true,
		},
		{
			name:    "chart as local ref error",
			chart:   "bad/path",
			version: "",
			wantErr: true,
			setupMock: func() {
				mockClient.EXPECT().
					GetChart("bad/path", gomock.Any()).
					Return(nil, "", errors.New("fail"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock()
			}
			r := resolver
			if tt.resolver != nil {
				r = tt.resolver
			}
			chart, ver, err := r.ResolveChartReference(tt.chart, tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantChart, chart)
				assert.Equal(t, tt.wantVer, ver)
			}
		})
	}
}
