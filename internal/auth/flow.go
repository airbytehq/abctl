package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/cli/browser"
	"github.com/google/uuid"
	"github.com/pterm/pterm"
)

//go:embed success.html
var successHTML string

const (
	// DefaultClientID is the default OAuth client ID for public clients
	DefaultClientID = "abctl"
	// CallbackPath is the OAuth callback path
	CallbackPath = "/callback"
)

// Flow manages the OAuth2/OIDC authorization flow.
// Uses channels and goroutines to handle async browser-based OAuth callbacks
// while maintaining a timeout. This pattern is necessary because we need to
// serve HTTP callbacks while the main thread waits for authentication to complete.
type Flow struct {
	Provider     *Provider
	ClientID     string
	RedirectPort int
	codeVerifier string
	state        string
}

// NewFlow creates a new OAuth flow with PKCE.
// The flow is designed to be reusable across different service contexts.
func NewFlow(provider *Provider, clientID string, callbackPort int) *Flow {
	return &Flow{
		Provider:     provider,
		ClientID:     clientID,
		RedirectPort: callbackPort,
		state:        uuid.New().String(),
		codeVerifier: generateCodeVerifier(),
	}
}

// Authenticate performs the OAuth flow and returns credentials.
// Uses goroutines and channels to handle the OAuth callback asynchronously
// because we need to run an HTTP server for the callback while also
// enforcing a timeout on the overall authentication process.
func (f *Flow) Authenticate(ctx context.Context) (*Credentials, error) {
	// Start callback server on fixed port
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", f.RedirectPort))
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server on port %d: %w", f.RedirectPort, err)
	}
	defer listener.Close()

	// Build and open authorization URL
	authURL := f.buildAuthURL()
	pterm.Info.Printf("Opening browser for authentication...\n")
	pterm.Info.Printf("If browser doesn't open, visit: %s\n", authURL)

	if err := browser.OpenURL(authURL); err != nil {
		pterm.Warning.Printf("Could not open browser automatically: %v\n", err)
	}

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
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timeout after 5 minutes")
	}
}

func (f *Flow) buildAuthURL() string {
	redirectURI := fmt.Sprintf("http://localhost:%d%s", f.RedirectPort, CallbackPath)
	codeChallenge := generateCodeChallenge(f.codeVerifier)

	params := url.Values{}
	params.Set("client_id", f.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", redirectURI)
	params.Set("state", f.state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("scope", "openid profile email offline_access")

	return fmt.Sprintf("%s?%s", f.Provider.AuthorizationEndpoint, params.Encode())
}

func (f *Flow) handleCallback(listener net.Listener, codeChan chan<- string, errChan chan<- error) {
	mux := http.NewServeMux()

	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		// Validate state
		if state := r.URL.Query().Get("state"); state != f.state {
			errChan <- fmt.Errorf("invalid state parameter")
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		// Check for errors
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
		fmt.Fprint(w, successHTML)

		// Send code
		codeChan <- code
	})

	server := &http.Server{Handler: mux}
	server.Serve(listener)
}

func (f *Flow) exchangeCode(ctx context.Context, code string) (*Credentials, error) {
	redirectURI := fmt.Sprintf("http://localhost:%d%s", f.RedirectPort, CallbackPath)

	tokens, err := ExchangeCodeForTokens(ctx, f.Provider, f.ClientID, code, redirectURI, f.codeVerifier)
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

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

