package api

import (
	"github.com/airbytehq/abctl/internal/http"
)

// Client handles Control Plane API operations
type Client struct {
	http http.HTTPDoer
}

// NewClient creates a new API client
func NewClient(httpDoer http.HTTPDoer) *Client {
	return &Client{
		http: httpDoer,
	}
}
