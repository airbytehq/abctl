package http

import (
	"net/http"
	"time"
)

// DefaultClient is the default HTTP client with reasonable timeout
var DefaultClient = &http.Client{
	Timeout: 30 * time.Second,
}