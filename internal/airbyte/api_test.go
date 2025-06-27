package airbyte

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const (
	host         = "https://localhost:1234"
	clientID     = "client-id"
	clientSecret = "client-secret"
)

func TestAirbyte_fetchToken(t *testing.T) {
	mockHTTP := &mockHTTPClient{}
	token := "token-test"
	airbyte := New(host, clientID, clientSecret, WithHTTPClient(mockHTTP))
	ctx := context.Background()

	t.Run("happy path", func(t *testing.T) {
		doCount := 0
		mockHTTP.do = func(req *http.Request) (*http.Response, error) {
			doCount++
			// check url
			if d := cmp.Diff(host+pathToken, req.URL.String()); d != "" {
				t.Errorf("unexpected request diff (-want +got):\n%s", d)
			}
			// check method
			if d := cmp.Diff(http.MethodPost, req.Method); d != "" {
				t.Errorf("unexpected request method (-want +got):\n%s", d)
			}
			// check headers
			if d := cmp.Diff("application/json", req.Header.Get("content-type")); d != "" {
				t.Errorf("unexpected request header content-type (-want +got):\n%s", d)
			}
			if d := cmp.Diff("application/json", req.Header.Get("accept")); d != "" {
				t.Errorf("unexpected request header accept (-want +got):\n%s", d)
			}
			// check body
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal("unable to read request body", err)
			}
			var (
				tokenRequestExpected = tokenRequest{
					GrantType:    grantType,
					ClientID:     clientID,
					ClientSecret: clientSecret,
				}
				tokenRequestActual tokenRequest
			)
			if err := json.Unmarshal(body, &tokenRequestActual); err != nil {
				t.Fatal("unable to parse request body", err)
			}
			if d := cmp.Diff(tokenRequestExpected, tokenRequestActual); d != "" {
				t.Errorf("unexpected token request diff (-want +got):\n%s", d)
			}

			res := tokenResponse{AccessToken: token}
			resBody, err := json.Marshal(res)
			if err != nil {
				t.Fatal("unable to marshal response body", err)
			}

			return &http.Response{
				Body: io.NopCloser(bytes.NewBuffer(resBody)),
			}, nil
		}

		tokenActual, err := airbyte.fetchToken(ctx)
		if err != nil {
			t.Fatal("unable to fetch token", err)
		}
		if d := cmp.Diff(Token(token), tokenActual); d != "" {
			t.Errorf("unexpected token response diff (-want +got):\n%s", d)
		}
		if d := cmp.Diff(1, doCount); d != "" {
			t.Errorf("unexpected request diff count (-want +got):\n%s", d)
		}

		// verify that another call to fetchToken doesn't call the http client (doCount doesn't change)
		tokenActual, err = airbyte.fetchToken(ctx)
		if err != nil {
			t.Fatal("unable to fetch token", err)
		}
		if d := cmp.Diff(Token(token), tokenActual); d != "" {
			t.Errorf("unexpected token response diff (-want +got):\n%s", d)
		}
		if d := cmp.Diff(1, doCount); d != "" {
			t.Errorf("unexpected request diff count (-want +got):\n%s", d)
		}
	})

	t.Run("error", func(t *testing.T) {
		airbyte.token = ""
		errTest := errors.New("test error")
		mockHTTP.do = func(req *http.Request) (*http.Response, error) {
			return nil, errTest
		}

		_, err := airbyte.fetchToken(ctx)
		if d := cmp.Diff(err, errTest, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error diff (-want +got):\n%s", d)
		}
	})
}

func TestAirbyte_GetOrgEmail(t *testing.T) {
	mockHTTP := &mockHTTPClient{}
	token := Token("token")
	airbyte := New(host, clientID, clientSecret, WithHTTPClient(mockHTTP), WithToken(token))
	ctx := context.Background()
	email := "test@example.test"

	t.Run("happy path", func(t *testing.T) {
		doCount := 0
		mockHTTP.do = func(req *http.Request) (*http.Response, error) {
			doCount++
			// check url
			if d := cmp.Diff(host+pathOrgGet, req.URL.String()); d != "" {
				t.Errorf("unexpected request diff (-want +got):\n%s", d)
			}
			// check method
			if d := cmp.Diff(http.MethodPost, req.Method); d != "" {
				t.Errorf("unexpected request method (-want +got):\n%s", d)
			}
			// check headers
			if d := cmp.Diff("application/json", req.Header.Get("content-type")); d != "" {
				t.Errorf("unexpected request header content-type (-want +got):\n%s", d)
			}
			if d := cmp.Diff("application/json", req.Header.Get("accept")); d != "" {
				t.Errorf("unexpected request header accept (-want +got):\n%s", d)
			}
			if d := cmp.Diff("Bearer "+string(token), req.Header.Get("Authorization")); d != "" {
				t.Errorf("unexpected request header authorization (-want +got):\n%s", d)
			}
			// check body
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal("unable to read request body", err)
			}
			var (
				emailRequestExpected = orgReq{OrgID: orgID}
				emailRequestActual   orgReq
			)
			if err := json.Unmarshal(body, &emailRequestActual); err != nil {
				t.Fatal("unable to parse request body", err)
			}
			if d := cmp.Diff(emailRequestExpected, emailRequestActual); d != "" {
				t.Errorf("unexpected token request diff (-want +got):\n%s", d)
			}

			res := organization{
				ID:    orgID,
				Name:  "test org",
				Email: email,
			}
			resBody, err := json.Marshal(res)
			if err != nil {
				t.Fatal("unable to marshal response body", err)
			}

			return &http.Response{Body: io.NopCloser(bytes.NewBuffer(resBody))}, nil
		}

		emailActual, err := airbyte.GetOrgEmail(ctx)
		if err != nil {
			t.Fatal("unable to fetch email", err)
		}
		if d := cmp.Diff(email, emailActual); d != "" {
			t.Errorf("unexpected email response diff (-want +got):\n%s", d)
		}
		if d := cmp.Diff(1, doCount); d != "" {
			t.Errorf("unexpected request diff count (-want +got):\n%s", d)
		}
	})

	t.Run("error", func(t *testing.T) {
		errTest := errors.New("test error")
		mockHTTP.do = func(req *http.Request) (*http.Response, error) {
			return nil, errTest
		}

		_, err := airbyte.GetOrgEmail(ctx)
		if d := cmp.Diff(err, errTest, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error (-want +got):\n%s", d)
		}
	})
}

func TestAirbyte_SetOrgEmail(t *testing.T) {
	mockHTTP := &mockHTTPClient{}
	token := Token("token")
	airbyte := New(host, clientID, clientSecret, WithHTTPClient(mockHTTP), WithToken(token))
	ctx := context.Background()
	email := "test@example.test"
	org := organization{
		ID:    orgID,
		Name:  "test org",
		Email: "test-orig@example.test",
	}

	t.Run("happy path", func(t *testing.T) {
		doCount := 0
		mockHTTP.do = func(req *http.Request) (*http.Response, error) {
			doCount++
			// first call is to get
			if doCount == 1 {
				if d := cmp.Diff(host+pathOrgGet, req.URL.String()); d != "" {
					t.Fatalf("unexpected request diff (-want +got):\n%s", d)
				}

				resBody, err := json.Marshal(org)
				if err != nil {
					t.Fatal("unable to marshal response body", err)
				}

				return &http.Response{Body: io.NopCloser(bytes.NewBuffer(resBody))}, nil
			}

			// check url
			if d := cmp.Diff(host+pathOrgSet, req.URL.String()); d != "" {
				t.Errorf("unexpected request diff (-want +got):\n%s", d)
			}
			// check method
			if d := cmp.Diff(http.MethodPost, req.Method); d != "" {
				t.Errorf("unexpected request method (-want +got):\n%s", d)
			}
			// check headers
			if d := cmp.Diff("application/json", req.Header.Get("content-type")); d != "" {
				t.Errorf("unexpected request header content-type (-want +got):\n%s", d)
			}
			if d := cmp.Diff("application/json", req.Header.Get("accept")); d != "" {
				t.Errorf("unexpected request header accept (-want +got):\n%s", d)
			}
			if d := cmp.Diff("Bearer "+string(token), req.Header.Get("Authorization")); d != "" {
				t.Errorf("unexpected request header authorization (-want +got):\n%s", d)
			}
			// check body
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal("unable to read request body", err)
			}
			var (
				emailRequestExpected = org
				emailRequestActual   organization
			)
			emailRequestExpected.Email = email

			if err := json.Unmarshal(body, &emailRequestActual); err != nil {
				t.Fatal("unable to parse request body", err)
			}
			if d := cmp.Diff(emailRequestExpected, emailRequestActual); d != "" {
				t.Errorf("unexpected token request diff (-want +got):\n%s", d)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer([]byte{})),
			}, nil
		}

		if err := airbyte.SetOrgEmail(ctx, email); err != nil {
			t.Fatal("unable to set email", err)
		}
		if d := cmp.Diff(2, doCount); d != "" {
			t.Errorf("unexpected request diff count (-want +got):\n%s", d)
		}
	})

	t.Run("error", func(t *testing.T) {
		errTest := errors.New("test error")
		doCount := 0

		mockHTTP.do = func(req *http.Request) (*http.Response, error) {
			doCount++
			// on the first call (the get call), return a response
			if doCount == 1 {
				resBody, err := json.Marshal(org)
				if err != nil {
					t.Fatal("unable to marshal response body", err)
				}

				return &http.Response{Body: io.NopCloser(bytes.NewBuffer(resBody))}, nil
			}

			// fail on the second call (the set call)
			return nil, errTest
		}

		err := airbyte.SetOrgEmail(ctx, email)
		if d := cmp.Diff(err, errTest, cmpopts.EquateErrors()); d != "" {
			t.Errorf("unexpected error (-want +got):\n%s", d)
		}
	})
}

// --- mocks

type mockHTTPClient struct {
	do func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.do(req)
}
