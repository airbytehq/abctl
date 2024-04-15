package telemetry

import (
	"airbyte.io/abctl/internal/build"
	"bytes"
	"fmt"
	"github.com/oklog/ulid/v2"
	"github.com/pbnjay/memory"
	"k8s.io/apimachinery/pkg/util/json"
	"maps"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

const (
	trackingKey = "kpYsVGLgxEqD5OuSZAQ9zWmdgBlyiaej"
	url         = "https://api.segment.io/v1/track"
)

var _ Client = (*SegmentClient)(nil)

// SegmentClient client, all methods communicate with segment.
type SegmentClient struct {
	h         http.Client
	sessionID ulid.ULID
	cfg       Config
	attrs     map[string]string
}

func NewSegmentClient(cfg Config) *SegmentClient {
	return &SegmentClient{
		h:         http.Client{Timeout: 10 * time.Second},
		cfg:       cfg,
		sessionID: ulid.Make(),
		attrs:     map[string]string{},
	}
}

func (s *SegmentClient) Start(et EventType) error {
	return s.send(Start, et, nil)
}

func (s *SegmentClient) Success(et EventType) error {
	return s.send(Success, et, nil)
}

func (s *SegmentClient) Failure(et EventType, err error) error {
	return s.send(Failed, et, err)
}

func (s *SegmentClient) Attr(key, val string) {
	s.attrs[key] = val
}

func (s *SegmentClient) send(es EventState, et EventType, ee error) error {
	properties := map[string]string{
		"deployment_method": "abctl",
		"session_id":        s.sessionID.String(),
		"state":             string(es),
		"os":                runtime.GOOS,
		"build":             build.Version,
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
		ID:         s.cfg.UserID.String(),
		Event:      string(et),
		Properties: properties,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		WriteKey:   trackingKey,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("could not create request body: %w", err)
	}

	resp, err := s.h.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("could not post: %w", err)
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
