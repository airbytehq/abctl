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
	// callbackPath is the OAuth callback path
	callbackPath = "/callback"
)

// Flow manages the OAuth2/OIDC authorization flow.
// Uses channels and goroutines to handle async browser-based OAuth callbacks
// while maintaining a timeout. This pattern is necessary because we need to
// serve HTTP callbacks while the main thread waits for authentication to complete.
type Flow struct {
	provider     *OIDCProvider
	clientID     string
	callbackPort int           // Port for our callback server (0 = random)
	SkipBrowser  bool          // Skip browser opening for tests
	timeout      time.Duration // Configurable timeout
	httpClient   http.HTTPDoer
	codeVerifier string
	state        string
	codeChan     chan string
	errChan      chan error
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

// WithProvider sets a specific OIDC provider for the flow
func WithProvider(provider *OIDCProvider) FlowOption {
	return func(f *Flow) {
		f.provider = provider
	}
}

// NewFlow creates a new OAuth flow with PKCE.
// The flow is designed to be reusable across different service contexts.
func NewFlow(clientID string, callbackPort int, httpClient http.HTTPDoer, opts ...FlowOption) *Flow {
	flow := &Flow{
		provider:     &OIDCProvider{},
		clientID:     clientID,
		callbackPort: callbackPort,
		timeout:      defaultAuthTimeout,
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
func (f *Flow) StartCallbackServer() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", f.callbackPort))
	if err != nil {
		return fmt.Errorf("failed to start callback server on port %d: %w", f.callbackPort, err)
	}

	// Update callbackPort with actual port if it was 0 (random port)
	if f.callbackPort == 0 {
		f.callbackPort = listener.Addr().(*net.TCPAddr).Port
	}

	// Initialize channels
	f.codeChan = make(chan string, 1)
	f.errChan = make(chan error, 1)

	// Start HTTP server in goroutine
	go f.handleCallback(listener, f.codeChan, f.errChan)

	return nil
}

// SendAuthRequest opens browser with auth URL
func (f *Flow) SendAuthRequest() error {
	authURL := f.buildAuthURL()
	if !f.SkipBrowser {
		_ = browser.OpenURL(authURL) // Best effort - user can manually open URL if browser fails
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
		_ = resp.Body.Close() // Best effort cleanup
	}

	return nil
}

// WaitForCallback waits for OAuth callback and returns credentials
func (f *Flow) WaitForCallback(ctx context.Context) (*Credentials, error) {
	// Wait for authorization code with timeout
	select {
	case code := <-f.codeChan:
		return f.exchangeCode(ctx, code)
	case err := <-f.errChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(f.timeout):
		return nil, fmt.Errorf("authentication timeout after %v", f.timeout)
	}
}

func (f *Flow) handleCallback(listener net.Listener, codeChan chan<- string, errChan chan<- error) {
	mux := stdhttp.NewServeMux()

	mux.HandleFunc(callbackPath, func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
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
		_, _ = fmt.Fprint(w, successHTML) // HTTP response write - handled by server

		// Send code
		codeChan <- code
	})

	server := &stdhttp.Server{Handler: mux}
	_ = server.Serve(listener) // Returns error when server shuts down, which is expected behavior
}

func (f *Flow) exchangeCode(ctx context.Context, code string) (*Credentials, error) {
	tokens, err := f.exchangeCodeForTokens(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for tokens: %w", err)
	}

	// Calculate expiration time
	if tokens.ExpiresIn <= 0 {
		return nil, fmt.Errorf("token without expiration is not allowed for security reasons")
	}

	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

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

func (f *Flow) buildAuthURL() string {
	handler := f.provider.AuthEndpointHandler()
	if handler == nil {
		return "" // FlowProvider doesn't support authorization endpoint
	}

	redirectURI := fmt.Sprintf("http://localhost:%d%s", f.callbackPort, callbackPath)
	codeChallenge := generateCodeChallenge(f.codeVerifier)

	return handler(f.clientID, redirectURI, f.state, codeChallenge)
}

// PKCE helpers

// DefaultStateGenerator generates a secure random OAuth state parameter for CSRF protection.
func DefaultStateGenerator() string {
	return uuid.New().String()
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // Cryptographic random never fails on supported platforms
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// ExchangeCodeForTokens exchanges an authorization code for tokens
func (f *Flow) exchangeCodeForTokens(ctx context.Context, code string) (*TokenResponse, error) {
	redirectURI := fmt.Sprintf("http://localhost:%d%s", f.callbackPort, callbackPath)

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", f.clientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	if f.codeVerifier != "" {
		data.Set("code_verifier", f.codeVerifier)
	}

	return doTokenRequest(ctx, f.provider.GetTokenEndpoint(), data, f.httpClient)
}
