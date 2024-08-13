package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	pathToken = "/api/v1/applications/token"
	pathUser  = "/api/public/v1/organizations"
)

var client httpClient = &http.Client{Timeout: 10 * time.Second}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func Email(ctx context.Context, host, clientID, clientSecret string) (string, error) {
	token, err := fetchToken(ctx, host, clientID, clientSecret)
	if err != nil {
		return "", fmt.Errorf("unable to fetch token: %w", err)
	}
	email, err := fetchEmail(ctx, host, token)

	if err != nil {
		return "", fmt.Errorf("unable to fetch email: %w", err)
	}

	return email, nil
}

const grantType = "client_credentials"

func fetchToken(ctx context.Context, host, clientID, clientSecret string) (string, error) {
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

	jsonData, err := json.Marshal(tokenRequest{GrantType: grantType, ClientID: clientID, ClientSecret: clientSecret})
	if err != nil {
		return "", fmt.Errorf("unable to marshal token request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host+pathToken, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("unable to create token request: %w", err)
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")

	res, err := client.Do(req)
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

	return tokenRes.AccessToken, nil
}

func fetchEmail(ctx context.Context, host, token string) (string, error) {
	type orgResponse struct {
		Data []struct {
			OrgID   string `json:"organizationId"`
			OrgName string `json:"organizationName"`
			Email   string `json:"email"`
		} `json:"data"`
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, host+pathUser, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create email request: %w", err)
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)

	res, err := client.Do(req)
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
