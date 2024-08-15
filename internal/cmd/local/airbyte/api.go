package airbyte

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	pathToken = "/api/v1/applications/token"
	pathUser  = "/api/public/v1/organizations"
	grantType = "client_credentials"
)

// Option for configuring the Command, primarily exists for testing
type Option func(*Airbyte)

type Token string

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Airbyte struct {
	h     HTTPClient
	token Token
	mu    sync.Mutex

	clientID     string
	clientSecret string
	host         string
}

func WithHTTPClient(h HTTPClient) Option {
	return func(a *Airbyte) {
		a.h = h
	}
}

// New returns an Airbyte client.
// The host is the hostname (with port) where the Airbyte API is hosted.
// The clientID and clientSecret are both required in order to create an application token.
func New(host, clientID, clientSecret string, opts ...Option) *Airbyte {
	a := &Airbyte{
		host:         host,
		clientID:     clientID,
		clientSecret: clientSecret,
	}
	for _, opt := range opts {
		opt(a)
	}
	if a.h == nil {
		a.h = &http.Client{Timeout: 10 * time.Second}
	}
	return a
}

// GetOrgEmail returns the organization email for the organization "00000000-0000-0000-0000-000000000000".
func (a *Airbyte) GetOrgEmail(ctx context.Context) (string, error) {
	token, err := a.fetchToken(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to determine application token: %w", err)
	}

	type orgResponse struct {
		Data []struct {
			OrgID   string `json:"organizationId"`
			OrgName string `json:"organizationName"`
			Email   string `json:"email"`
		} `json:"data"`
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.host+pathUser, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create email request: %w", err)
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+string(token))

	res, err := a.h.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to send email request: %w", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read email response: %w", err)
	}

	var orgRes orgResponse
	if err := json.Unmarshal(resBody, &orgRes); err != nil {
		return "", fmt.Errorf("unable to decode email request: %w", err)
	}

	if len(orgRes.Data) < 1 {
		return "", errors.New("unable to find any organizations")
	}

	if orgRes.Data[0].OrgID != "00000000-0000-0000-0000-000000000000" {
		return "", fmt.Errorf("could not find expected organization, found %s", orgRes.Data[0].OrgID)
	}

	return orgRes.Data[0].Email, nil
}

func (a *Airbyte) fetchToken(ctx context.Context) (Token, error) {
	t := a.token
	if t != "" {
		return t, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// check again if there is now a token
	if a.token != "" {
		return a.token, nil
	}

	type (
		tokenResponse struct {
			AccessToken string `json:"access_token"`
		}
		tokenRequest struct {
			GrantType    string `json:"grant_type"`
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		}
	)

	jsonData, err := json.Marshal(tokenRequest{GrantType: grantType, ClientID: a.clientID, ClientSecret: a.clientSecret})
	if err != nil {
		return "", fmt.Errorf("unable to marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.host+pathToken, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("unable to create token request: %w", err)
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")

	res, err := a.h.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to send token request: %w", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read token response: %w", err)
	}

	var tokenRes tokenResponse
	if err := json.Unmarshal(resBody, &tokenRes); err != nil {
		return "", fmt.Errorf("unable to decode token request: %w", err)
	}

	a.token = Token(tokenRes.AccessToken)
	return a.token, nil
}
