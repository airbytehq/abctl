package telemetry

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/google/uuid"
	"github.com/pbnjay/memory"
	"github.com/pterm/pterm"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ Client = (*SegmentClient)(nil)

// Doer interface for testing purposes
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Option is a configuration option for segment
type Option func(*SegmentClient)

// WithHTTPClient overrides the default http.Client, primarily for testing purposes.
func WithHTTPClient(d Doer) Option {
	return func(client *SegmentClient) {
		client.doer = d
	}
}

// WithSessionID overrides the default ulid session, primarily for testing purposes.
func WithSessionID(sessionID uuid.UUID) Option {
	return func(client *SegmentClient) {
		client.sessionID = sessionID
	}
}

var _ Client = (*SegmentClient)(nil)

// SegmentClient client, all methods communicate with segment.
type SegmentClient struct {
	doer      Doer
	sessionID uuid.UUID
	cfg       Config
	attrs     map[string]string
}

func NewSegmentClient(cfg Config, opts ...Option) *SegmentClient {
	cli := &SegmentClient{
		doer:      &http.Client{Timeout: 10 * time.Second},
		cfg:       cfg,
		sessionID: uuid.New(),
		attrs:     map[string]string{},
	}

	for _, opt := range opts {
		opt(cli)
	}

	return cli
}

func (s *SegmentClient) Start(ctx context.Context, et EventType) error {
	return s.send(ctx, Start, et, nil)
}

func (s *SegmentClient) Success(ctx context.Context, et EventType) error {
	return s.send(ctx, Success, et, nil)
}

func (s *SegmentClient) Failure(ctx context.Context, et EventType, err error) error {
	return s.send(ctx, Failed, et, err)
}

func (s *SegmentClient) Attr(key, val string) {
	s.attrs[key] = val
}

func (s *SegmentClient) User() uuid.UUID {
	return s.cfg.AnalyticsID.toUUID()
}

func (s *SegmentClient) Wrap(ctx context.Context, et EventType, f func() error) error {
	attemptSuccessFailure := true

	if err := s.Start(ctx, et); err != nil {
		pterm.Debug.Printfln("Unable to send telemetry start data: %s", err)
		attemptSuccessFailure = false
	}

	if err := f(); err != nil {
		if attemptSuccessFailure {
			if errTel := s.Failure(ctx, et, err); errTel != nil {
				pterm.Debug.Printfln("Unable to send telemetry failure data: %s", errTel)
			}
		}

		return err
	}

	if attemptSuccessFailure {
		if err := s.Success(ctx, et); err != nil {
			pterm.Debug.Printfln("Unable to send telemetry success data: %s", err)
		}
	}

	return nil
}

const (
	trackingKey = "kpYsVGLgxEqD5OuSZAQ9zWmdgBlyiaej"
	url         = "https://api.segment.io/v1/track"
)

func (s *SegmentClient) send(ctx context.Context, es EventState, et EventType, ee error) error {
	properties := map[string]string{
		"deployment_method": "abctl",
		"session_id":        s.sessionID.String(),
		"state":             string(es),
		"os":                runtime.GOOS,
		"build":             build.Version,
		"script_version":    build.Version,
		"cpu_count":         strconv.Itoa(runtime.NumCPU()),
		"mem_total_bytes":   strconv.FormatUint(memory.TotalMemory(), 10),
		"mem_free_bytes":    strconv.FormatUint(memory.FreeMemory(), 10),
	}
	// add all the attributes to the properties map before sending it
	maps.Copy(properties, s.attrs)

	if ee != nil {
		properties["error"] = ee.Error()
	}

	body := body{
		ID:         s.cfg.AnalyticsID.String(),
		Event:      string(et),
		Properties: properties,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		WriteKey:   trackingKey,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("unable to create request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("unable to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.doer.Do(req)
	if err != nil {
		return fmt.Errorf("unable to post: %w", err)
	}

	defer resp.Body.Close()
	return nil
}

type body struct {
	ID         string            `json:"anonymousId"`
	Event      string            `json:"event"`
	Properties map[string]string `json:"properties"`
	Timestamp  string            `json:"timestamp"`
	WriteKey   string            `json:"writeKey"`
}
