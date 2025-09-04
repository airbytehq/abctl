package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"net"
	stdhttp "net/http"
	"net/url"
	"time"

	"github.com/airbytehq/abctl/internal/http"
	"github.com/cli/browser"
	"github.com/google/uuid"
)

//go:embed success.html
var successHTML string

const (
	// CallbackPath is the OAuth callback path
	CallbackPath = "/callback"
)

// Flow manages the OAuth2/OIDC authorization flow.
// Uses channels and goroutines to handle async browser-based OAuth callbacks
// while maintaining a timeout. This pattern is necessary because we need to
// serve HTTP callbacks while the main thread waits for authentication to complete.
type Flow struct {
	provider     *Provider
	clientID     string
	redirectPort int
	SkipBrowser  bool          // Skip browser opening for tests
	Timeout      time.Duration // Configurable timeout (default 5 minutes)
	httpClient   http.HTTPDoer
	codeVerifier string
	state        string
}

// FlowOption is used to configure the flow object.
type FlowOption func(f *Flow)

// StateGenerator is a function that generates the OAuth state parameter for CSRF protection.
type StateGenerator func() string

// WithStateGenerator is an option used to override the state generation for authentication.
func WithStateGenerator(g StateGenerator) FlowOption {
	return func(f *Flow) {
		f.state = g()
	}
}

// NewFlow creates a new OAuth flow with PKCE.
// The flow is designed to be reusable across different service contexts.
func NewFlow(provider *Provider, clientID string, callbackPort int, httpClient http.HTTPDoer, opts ...FlowOption) *Flow {
	flow := &Flow{
		provider:     provider,
		clientID:     clientID,
		redirectPort: callbackPort,
		Timeout:      5 * time.Minute, // Default timeout
		httpClient:   httpClient,
		state:        "", // Set by options or DefaultStateGenerator
		codeVerifier: generateCodeVerifier(),
	}

	// Apply any flow options
	for _, opt := range opts {
		if opt != nil {
			opt(flow)
		}
	}

	// If no state generator was provided via options, use default
	if flow.state == "" {
		flow.state = DefaultStateGenerator()
	}

	return flow
}

// GetAuthURL returns the authorization URL for manual browser opening
func (f *Flow) GetAuthURL() string {
	return f.buildAuthURL()
}

// StartCallbackServer starts the OAuth callback server
func (f *Flow) StartCallbackServer() (net.Listener, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", f.redirectPort))
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server on port %d: %w", f.redirectPort, err)
	}
	return listener, nil
}

// SendAuthRequest opens browser with auth URL
func (f *Flow) SendAuthRequest() error {
	authURL := f.buildAuthURL()
	if !f.SkipBrowser {
		browser.OpenURL(authURL)
	}

	// Also make HTTP request for tests
	u, err := url.Parse(authURL)
	if err != nil {
		return fmt.Errorf("failed to parse auth URL: %w", err)
	}

	resp, err := f.httpClient.Do(&stdhttp.Request{
		Method: "GET",
		URL:    u,
	})
	if err != nil {
		return fmt.Errorf("failed to send auth request: %w", err)
	}

	if resp != nil {
		resp.Body.Close()
	}

	return nil
}

// WaitForCallback waits for OAuth callback and returns credentials
func (f *Flow) WaitForCallback(ctx context.Context, listener net.Listener) (*Credentials, error) {
	// Wait for callback
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go f.handleCallback(listener, codeChan, errChan)

	// Wait for authorization code with timeout
	select {
	case code := <-codeChan:
		return f.exchangeCode(ctx, code)
	case err := <-errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(f.Timeout):
		return nil, fmt.Errorf("authentication timeout after %v", f.Timeout)
	}
}

func (f *Flow) buildAuthURL() string {
	redirectURI := fmt.Sprintf("http://localhost:%d%s", f.redirectPort, CallbackPath)
	codeChallenge := generateCodeChallenge(f.codeVerifier)

	params := url.Values{}
	params.Set("client_id", f.clientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", redirectURI)
	params.Set("state", f.state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("scope", "openid profile email offline_access")

	return fmt.Sprintf("%s?%s", f.provider.AuthorizationEndpoint, params.Encode())
}

func (f *Flow) handleCallback(listener net.Listener, codeChan chan<- string, errChan chan<- error) {
	mux := stdhttp.NewServeMux()

	mux.HandleFunc(CallbackPath, func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		// Validate state
		if state := r.URL.Query().Get("state"); state != f.state {
			errChan <- fmt.Errorf("invalid state parameter")
			stdhttp.Error(w, "Invalid state", stdhttp.StatusBadRequest)
			return
		}

		// Check for errors
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errDesc := r.URL.Query().Get("error_description")
			errChan <- fmt.Errorf("authentication error: %s - %s", errParam, errDesc)
			stdhttp.Error(w, "Authentication failed", stdhttp.StatusBadRequest)
			return
		}

		// Get authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			stdhttp.Error(w, "No code received", stdhttp.StatusBadRequest)
			return
		}

		// Send success response
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, successHTML)

		// Send code
		codeChan <- code
	})

	server := &stdhttp.Server{Handler: mux}
	server.Serve(listener)
}

func (f *Flow) exchangeCode(ctx context.Context, code string) (*Credentials, error) {
	redirectURI := fmt.Sprintf("http://localhost:%d%s", f.redirectPort, CallbackPath)

	tokens, err := ExchangeCodeForTokens(ctx, f.provider, f.clientID, code, redirectURI, f.codeVerifier, f.httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for tokens: %w", err)
	}

	// Calculate expiration time
	expiresAt := time.Now()
	if tokens.ExpiresIn > 0 {
		expiresAt = expiresAt.Add(time.Duration(tokens.ExpiresIn) * time.Second)
	} else {
		// Default to 1 hour if not specified
		expiresAt = expiresAt.Add(time.Hour)
	}

	tokenType := tokens.TokenType
	if tokenType == "" {
		tokenType = "Bearer" // Default to Bearer if not specified
	}

	return &Credentials{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokenType,
		ExpiresAt:    expiresAt,
	}, nil
}

// PKCE helpers

// DefaultStateGenerator generates a secure random OAuth state parameter for CSRF protection.
func DefaultStateGenerator() string {
	return uuid.New().String()
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
