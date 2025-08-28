package api

import (
	"net/http"
)

// Client handles Control Plane API operations
type Client struct {
	http HTTPDoer
}

// HTTPDoer interface for making HTTP requests
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewClient creates a new API client
func NewClient(http HTTPDoer) *Client {
	return &Client{
		http: http,
	}
}
