package airbox

import (
	"fmt"
	"net/url"

	"github.com/airbytehq/abctl/internal/auth"
)

// NewConfigInitError returns an error with config init instructions for the given message.
func NewConfigInitError(msg string) error {
	return fmt.Errorf("%s - please run 'airbox config init' first", msg)
}

// NewLoginError returns an error with login instructions for the given message.
func NewLoginError(msg string) error {
	return fmt.Errorf("%s - please run 'airbox auth login' first", msg)
}

// Config represents the airbox configuration file structure.
type Config struct {
	CurrentContext string            `json:"current-context" yaml:"current-context"`
	Credentials    *auth.Credentials `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	Contexts       []NamedContext    `json:"contexts" yaml:"contexts"`
}

// NamedContext represents a named context entry.
type NamedContext struct {
	Name    string  `json:"name" yaml:"name"`
	Context Context `json:"context" yaml:"context"`
}

// Context represents a single airbox context (org/workspace combination).
type Context struct {
	AirbyteAPIURL  string `json:"airbyteApiUrl" yaml:"airbyteApiUrl"`
	AirbyteURL     string `json:"airbyteUrl" yaml:"airbyteUrl"`
	OrganizationID string `json:"organizationId" yaml:"organizationId"`
	Edition        string `json:"edition" yaml:"edition"`
	Auth           Auth   `json:"auth" yaml:"auth"`
}

// Validate ensures the context has all required fields and valid URLs.
func (c *Context) Validate() error {
	if c.AirbyteAPIURL == "" {
		return fmt.Errorf("airbyteApiUrl is required")
	}
	if _, err := url.Parse(c.AirbyteAPIURL); err != nil {
		return fmt.Errorf("invalid airbyteApiUrl: %w", err)
	}

	if c.AirbyteURL == "" {
		return fmt.Errorf("airbyteUrl is required")
	}
	if _, err := url.Parse(c.AirbyteURL); err != nil {
		return fmt.Errorf("invalid airbyteUrl: %w", err)
	}

	if c.Edition == "" {
		return fmt.Errorf("edition is required")
	}

	return c.Auth.Validate()
}

// GetCurrentContext returns the current context.
func (c *Config) GetCurrentContext() (*Context, error) {
	if c.CurrentContext == "" {
		return nil, fmt.Errorf("no current context set")
	}

	for _, namedContext := range c.Contexts {
		if namedContext.Name == c.CurrentContext {
			return &namedContext.Context, nil
		}
	}

	return nil, fmt.Errorf("current context %q not found", c.CurrentContext)
}

// SetCurrentContext sets the current context
func (c *Config) SetCurrentContext(name string) error {
	// Verify context exists
	for _, namedContext := range c.Contexts {
		if namedContext.Name == name {
			c.CurrentContext = name
			return nil
		}
	}

	return fmt.Errorf("context %q not found", name)
}

// AddContext adds or updates a context
func (c *Config) AddContext(name string, context Context) {
	// Check if context already exists
	for i, namedContext := range c.Contexts {
		if namedContext.Name == name {
			c.Contexts[i].Context = context
			return
		}
	}

	// Add new context
	c.Contexts = append(c.Contexts, NamedContext{
		Name:    name,
		Context: context,
	})

	// Set as current if it's the first context
	if c.CurrentContext == "" {
		c.CurrentContext = name
	}
}

// RemoveContext removes a context
func (c *Config) RemoveContext(name string) error {
	for i, namedContext := range c.Contexts {
		if namedContext.Name == name {
			c.Contexts = append(c.Contexts[:i], c.Contexts[i+1:]...)

			// If this was the current context, clear it
			if c.CurrentContext == name {
				if len(c.Contexts) > 0 {
					c.CurrentContext = c.Contexts[0].Name
				} else {
					c.CurrentContext = ""
				}
			}
			return nil
		}
	}

	return fmt.Errorf("context %q not found", name)
}

// GetCredentials returns the current user credentials
func (c *Config) GetCredentials() (*auth.Credentials, error) {
	if c.Credentials == nil {
		return nil, fmt.Errorf("no user credentials found")
	}
	return c.Credentials, nil
}

// IsAuthenticated checks if user has credentials (expiry is handled by auth client)
func (c *Config) IsAuthenticated() bool {
	return c.Credentials != nil && c.Credentials.AccessToken != ""
}

// SetCredentials updates the user credentials in the config (does not save)
func (c *Config) SetCredentials(creds *auth.Credentials) {
	c.Credentials = creds
}

// Validate ensures the config structure is valid and coherent.
func (c *Config) Validate() error {
	// Check if current context exists in contexts list
	if c.CurrentContext != "" {
		found := false
		for _, namedContext := range c.Contexts {
			if namedContext.Name == c.CurrentContext {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("current context %q not found in contexts list", c.CurrentContext)
		}
	}

	// Check for duplicate context names
	seen := make(map[string]bool)
	for _, namedContext := range c.Contexts {
		if namedContext.Name == "" {
			return fmt.Errorf("context name cannot be empty")
		}
		if seen[namedContext.Name] {
			return fmt.Errorf("duplicate context name: %q", namedContext.Name)
		}
		seen[namedContext.Name] = true

		// Validate each context
		if err := namedContext.Context.Validate(); err != nil {
			return fmt.Errorf("invalid context %q: %w", namedContext.Name, err)
		}
	}

	// Validate credentials if present
	if c.Credentials != nil {
		if c.Credentials.AccessToken == "" {
			return fmt.Errorf("credentials present but access token is empty")
		}
	}

	return nil
}
