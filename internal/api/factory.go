package api

import (
	"context"

	"github.com/airbytehq/abctl/internal/airbox"
	"github.com/airbytehq/abctl/internal/http"
)

// Factory creates authenticated API clients
type Factory func(ctx context.Context, cfg airbox.ConfigProvider, client http.HTTPDoer) (*Client, error)

// NewFactory creates an authenticated API client
func NewFactory(ctx context.Context, cfg airbox.ConfigProvider, httpDoer http.HTTPDoer) (*Client, error) {
	authedHTTP, err := airbox.CreateHTTPClient(ctx, cfg, httpDoer)
	if err != nil {
		return nil, err
	}
	return NewClient(authedHTTP), nil
}
