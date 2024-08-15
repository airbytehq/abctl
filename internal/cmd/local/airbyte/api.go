package airbyte

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	pathToken  = "/api/v1/applications/token"
	pathOrgGet = "/api/v1/organizations/get"
	pathOrgSet = "/api/v1/organizations/update"
	grantType  = "client_credentials"
)

// Token represents an application token
type Token string

// Option for configuring the Command, primarily exists for testing
type Option func(*Airbyte)

// HTTPClient exists for testing purposes
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Airbyte is used for communicating with the Airbyte API
type Airbyte struct {
	h     HTTPClient
	token Token
	mu    sync.Mutex

	clientID     string
	clientSecret string
	host         string
}

// WithHTTPClient overrides the default http client.
// Primarily for testing purposes.
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
	org, err := a.getOrg(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get organization: %w", err)
	}

	return org.Email, nil
}

// SetOrgEmail updates the email assocated with the default organization "00000000-0000-0000-0000-000000000000".
func (a *Airbyte) SetOrgEmail(ctx context.Context, email string) error {
	org, err := a.getOrg(ctx)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	org.Email = email

	if err := a.setOrg(ctx, org); err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	return nil
}

// getOrg returns the default organization "00000000-0000-0000-0000-000000000000".
func (a *Airbyte) getOrg(ctx context.Context) (organization, error) {
	token, err := a.fetchToken(ctx)
	if err != nil {
		return organization{}, fmt.Errorf("unable to fetch token: %w", err)
	}

	type orgReq struct {
		OrgID string `json:"organizationId"`
	}

	jsonData, err := json.Marshal(orgReq{OrgID: "00000000-0000-0000-0000-000000000000"})
	if err != nil {
		return organization{}, fmt.Errorf("unable to marshal organization request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.host+pathOrgGet, bytes.NewBuffer(jsonData))
	if err != nil {
		return organization{}, fmt.Errorf("unable to create organization request: %w", err)
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+string(token))

	res, err := a.h.Do(req)
	if err != nil {
		fmt.Errorf("unable to send organization request: %w", err)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return organization{}, fmt.Errorf("unable to read organization response: %w", err)
	}

	var org organization
	if err := json.Unmarshal(resBody, &org); err != nil {
		return organization{}, fmt.Errorf("unable to decode organization request: %w", err)
	}
	return org, nil
}

// setOrg calls the organization/update endpoint.
// This is a POST endpoint and does not support PATCH operations.
// Make sure the org provides has all of its attributes defined.
func (a *Airbyte) setOrg(ctx context.Context, org organization) error {
	token, err := a.fetchToken(ctx)
	if err != nil {
		return fmt.Errorf("unable to fetch token: %w", err)
	}

	jsonData, err := json.Marshal(org)
	if err != nil {
		return fmt.Errorf("unable to marshal organization request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.host+pathOrgSet, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("unable to create organization request: %w", err)
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+string(token))

	res, err := a.h.Do(req)
	if err != nil {
		fmt.Errorf("unable to send organization request: %w", err)
	}
	_ = res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}

// fetchToken returns the application token for the ClientID and ClientSecret.
// The token will be cached when first called, returning the cached value for every following call.
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

// organization is how the API models the Organization concept
type organization struct {
	ID       string `json:"organizationId"`
	Name     string `json:"organizationName"`
	Email    string `json:"email"`
	PBA      bool   `json:"pba"`
	Billing  bool   `json:"orgLevelBilling"`
	SSORealm string `json:"ssoRealm"`
}
