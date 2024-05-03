package update

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name    string
		remote  string
		local   string
		want    string
		wantErr error
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
		{
			name:   "local version is the same",
			remote: remoteVersion("v0.3.0"),
			local:  "v0.3.0",
			want:   "",
		},
		{
			name:    "no check if version is dev",
			local:   "dev",
			want:    "",
			wantErr: ErrDevVersion,
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
			if d := cmp.Diff(tt.wantErr, err, cmpopts.EquateErrors()); d != "" {
				t.Errorf("unexpected error: %s", err)
			}
			if d := cmp.Diff(tt.want, latest); d != "" {
				t.Errorf("unexpected diff (-want, +got) = %s", d)
			}
		})
	}
}

func TestCheck_HTTPRequest(t *testing.T) {
	var actualRequest *http.Request

	h := mockDoer{
		do: func(req *http.Request) (*http.Response, error) {
			actualRequest = req
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(remoteVersion("v0.1.0"))),
			}, nil
		},
	}

	if _, err := Check(context.Background(), h, "v0.1.0"); err != nil {
		t.Error("unexpected error:", err)
	}
	// verify method
	if d := cmp.Diff(http.MethodGet, actualRequest.Method); d != "" {
		t.Errorf("unexpected method (-want, +got) = %s", d)
	}
	// verify url
	if d := cmp.Diff(url, actualRequest.URL.String()); d != "" {
		t.Errorf("unexpected url (-want, +got) = %s", d)
	}
}

func TestCheck_HTTPErr(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		err    error
	}{
		{
			name:   "404 status",
			status: http.StatusNotFound,
		},
		{
			name:   "418 status",
			status: http.StatusTeapot,
		},
		{
			name:   "500 status",
			status: http.StatusInternalServerError,
		},
		{
			name:   "invalid json",
			status: http.StatusOK,
			body:   "",
		},
		{
			name:   "empty json",
			status: http.StatusOK,
			body:   "{}",
		},
		{
			name:   "empty version",
			status: http.StatusOK,
			body:   `{"tag_name":""}`,
		},
		{
			name:   "do returns error",
			status: http.StatusBadGateway,
			err:    errors.New("test error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := mockDoer{
				do: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.status,
						Body:       io.NopCloser(strings.NewReader(tt.body)),
					}, tt.err
				},
			}

			_, err := Check(context.Background(), h, "v0.1.0")
			if err == nil {
				t.Error("unexpected success")
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
