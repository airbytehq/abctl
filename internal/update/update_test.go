package update

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		local  string
		want   string
	}{
		{
			name:   "local version is newer",
			remote: remoteVersion("v0.1.0"),
			local:  "v0.2.0",
		},
		{
			name:   "local version is older",
			remote: remoteVersion("v0.2.0"),
			local:  "v0.1.0",
			want:   "v0.2.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h := mockDoer{
				do: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(tt.remote)),
					}, nil
				},
			}

			latest, err := Check(ctx, h, tt.local)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			if d := cmp.Diff(latest, tt.want); d != "" {
				t.Errorf("unexpected diff (-want, +got) = %s", d)
			}
		})
	}
}

func remoteVersion(version string) string {
	return fmt.Sprintf(`{ "tag_name": "%s" }`, version)
}

var _ doer = (*mockDoer)(nil)

type mockDoer struct {
	do func(req *http.Request) (*http.Response, error)
}

func (m mockDoer) Do(req *http.Request) (*http.Response, error) {
	return m.do(req)
}
