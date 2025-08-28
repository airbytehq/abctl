package http

import (
	"net/http"
	"net/url"
)

// Client handles HTTP requests with base URL resolution
type Client struct {
	doer    HTTPDoer
	baseURL *url.URL
}

// HTTPDoer interface for making HTTP requests
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewClient creates an HTTP client with any HTTPDoer implementation
func NewClient(baseURL string, doer HTTPDoer) (*Client, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		doer:    doer,
		baseURL: parsedURL,
	}, nil
}

// Do performs an HTTP request, prepending base URL
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	fullURL := c.baseURL.ResolveReference(req.URL)

	newReq := &http.Request{
		Method: req.Method,
		URL:    fullURL,
		Header: req.Header,
		Body:   req.Body,
	}

	if req.Context() != nil {
		newReq = newReq.WithContext(req.Context())
	}

	return c.doer.Do(newReq)
}
