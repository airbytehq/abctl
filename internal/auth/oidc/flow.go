package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cli/browser"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
)

// AuthFlow handles the OIDC authorization flow
type AuthFlow struct {
	Provider      *ProviderConfig
	ClientID      string
	RedirectPort  int
	PKCEChallenge *PKCEChallenge
	State         string
}

// NewAuthFlow creates a new authorization flow
func NewAuthFlow(provider *ProviderConfig, clientID string) (*AuthFlow, error) {
	pkce, err := GeneratePKCEChallenge()
	if err != nil {
		return nil, err
	}

	return &AuthFlow{
		Provider:      provider,
		ClientID:      clientID,
		PKCEChallenge: pkce,
		State:         uuid.New().String(),
		RedirectPort:  8085, // Default port, will find available one
	}, nil
}

// Authenticate performs the full authentication flow
func (f *AuthFlow) Authenticate(ctx context.Context) (*TokenResponse, error) {
	// Start local server for callback
	listener, err := f.startCallbackServer()
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer listener.Close()

	// Update redirect port based on actual listener
	addr := listener.Addr().(*net.TCPAddr)
	f.RedirectPort = addr.Port

	// Build authorization URL
	authURL := f.buildAuthorizationURL()

	// Open browser
	pterm.Info.Printf("Opening browser for authentication...\n")
	pterm.Info.Printf("If browser doesn't open, visit: %s\n", authURL)

	if err := browser.OpenURL(authURL); err != nil {
		pterm.Warning.Printf("Failed to open browser: %v\n", err)
	}

	// Wait for callback
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go f.handleCallback(listener, codeChan, errChan)

	select {
	case code := <-codeChan:
		// Exchange code for tokens
		return f.exchangeCodeForTokens(ctx, code)
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timeout")
	}
}

func (f *AuthFlow) buildAuthorizationURL() string {
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", f.RedirectPort)

	params := url.Values{}
	params.Set("client_id", f.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", redirectURI)
	params.Set("state", f.State)
	params.Set("code_challenge", f.PKCEChallenge.Challenge)
	params.Set("code_challenge_method", f.PKCEChallenge.Method)
	params.Set("scope", "openid")

	return fmt.Sprintf("%s?%s", f.Provider.AuthorizationEndpoint, params.Encode())
}

func (f *AuthFlow) startCallbackServer() (net.Listener, error) {
	// Try to listen on preferred port, then try random ports
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", f.RedirectPort))
	if err != nil {
		// Try random port
		listener, err = net.Listen("tcp", "localhost:0")
		if err != nil {
			return nil, err
		}
	}
	return listener, nil
}

func (f *AuthFlow) handleCallback(listener net.Listener, codeChan chan<- string, errChan chan<- error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Check state
		state := r.URL.Query().Get("state")
		if state != f.State {
			errChan <- fmt.Errorf("invalid state parameter")
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		// Check for error
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			errChan <- fmt.Errorf("authentication error: %s - %s", errParam, errDesc)
			http.Error(w, "Authentication failed", http.StatusBadRequest)
			return
		}

		// Get authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		// Send success response
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
			<body>
				<h1>Authentication Successful</h1>
				<p>You can close this window and return to the terminal.</p>
				<script>window.close();</script>
			</body>
			</html>
		`)

		// Send code to channel
		codeChan <- code
	})

	server := &http.Server{Handler: mux}
	_ = server.Serve(listener)
}

func (f *AuthFlow) exchangeCodeForTokens(ctx context.Context, code string) (*TokenResponse, error) {
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", f.RedirectPort)

	// Build form data
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", f.ClientID)
	data.Set("code_verifier", f.PKCEChallenge.Verifier)

	req, err := http.NewRequestWithContext(ctx, "POST", f.Provider.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: status %d", resp.StatusCode)
	}

	var tokens TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokens, nil
}

// RefreshToken refreshes an access token using a refresh token
func RefreshToken(ctx context.Context, provider *ProviderConfig, clientID, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)

	req, err := http.NewRequestWithContext(ctx, "POST", provider.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed: status %d", resp.StatusCode)
	}

	var tokens TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to decode refresh response: %w", err)
	}

	return &tokens, nil
}
