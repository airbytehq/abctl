package http

import (
	"fmt"
	"net/http"
	"net/url"
)

// Client handles HTTP requests with base URL resolution
type Client struct {
	doer    HTTPDoer
	baseURL string
}

// HTTPDoer interface for making HTTP requests
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewClient creates an HTTP client with any HTTPDoer implementation
func NewClient(baseURL string, doer HTTPDoer) (*Client, error) {
	return &Client{
		doer:    doer,
		baseURL: baseURL,
	}, nil
}

// Do performs an HTTP request, prepending base URL
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.doer == nil {
		return nil, fmt.Errorf("nil pointer dereference: doer is nil")
	}

	fullURLStr, err := url.JoinPath(c.baseURL, req.URL.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to join API base URL with request path: %w", err)
	}

	fullURL, err := url.Parse(fullURLStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Preserve the original query string
	fullURL.RawQuery = req.URL.RawQuery

	newReq := &http.Request{
		Method: req.Method,
		URL:    fullURL,
		Header: req.Header,
		Body:   req.Body,
	}

	// Preserve the context.
	if req.Context() != nil {
		newReq = newReq.WithContext(req.Context())
	}

	return c.doer.Do(newReq)
}
