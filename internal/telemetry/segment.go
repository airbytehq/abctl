package telemetry

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"path/filepath"
	"runtime"
	"time"
)

type EventState string

const (
	Start   EventState = "started"
	Failed  EventState = "failed"
	Success EventState = "succeeded"
)

type EventType string

const (
	Install  EventType = "install"
	Sync     EventType = "sync"
	Validate EventType = "validate"
	Check    EventType = "check"
)

const (
	trackingKey = "kpYsVGLgxEqD5OuSZAQ9zWmdgBlyiaej"
	url         = "https://api.segment.io/v1/track"
)

// TODO: complete analytics file support
var analyticsFile = filepath.Join(".airbyte", "analytics.yml")

type Client struct {
	h  http.Client
	id uuid.UUID
}

func New(id uuid.UUID) *Client {
	return &Client{
		h:  http.Client{Timeout: 10 * time.Second},
		id: id,
	}
}

func (c *Client) Start() error {
	return c.send(Start, Install, nil)
}

func (c *Client) Success() error {
	return c.send(Success, Install, nil)
}

func (c *Client) Failure(err error) error {
	return c.send(Failed, Install, err)
}

func (c *Client) send(es EventState, et EventType, ee error) error {
	body := body{
		ID:    "",
		Event: string(et),
		Properties: map[string]string{
			"session_id": c.id.String(),
			"state":      string(es),
			"os":         runtime.GOOS,
			// TODO: add k8s version, docker version, other?
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		WriteKey:  trackingKey,
	}
	if ee != nil {
		body.Error = ee.Error()
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("could not create request body: %w", err)
	}

	resp, err := c.h.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("could not post: %w", err)
	}

	defer resp.Body.Close()
	return nil
}

type body struct {
	ID         string            `json:"anonymousId"`
	Error      string            `json:"error,omitempty"`
	Event      string            `json:"event"`
	Properties map[string]string `json:"properties"`
	Timestamp  string            `json:"timestamp"`
	WriteKey   string            `json:"writeKey"`
}
