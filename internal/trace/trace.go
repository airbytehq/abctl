package trace

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/airbytehq/abctl/internal/cmd/local/paths"
	"github.com/airbytehq/abctl/internal/telemetry"
	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	"github.com/pterm/pterm"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// traceName is the name of the otel tracer
	tracerName = "github.com/airbytehq/abctl/trace"
	// redactedUserHome is the redacted user home directory
	redactedUserHome = "[USER_HOME]"
)

var (
	// May not be required, it is unclear if a tracer should be instantiated more than once.
	once   sync.Once
	tracer trace.Tracer
)

// NewSpan initializes the otel tracer, if necessary, and starts a new span with
// the provided name.  The returned span will be added to the returned context.
func NewSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	once.Do(func() {
		tracer = otel.Tracer(tracerName)
	})
	return tracer.Start(ctx, name)
}

// AttachLog attaches a log with the provided name and body.
func AttachLog(name, body string) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.AddAttachment(&sentry.Attachment{
			Filename:    name,
			ContentType: "test/plain",
			Payload:     []byte(body),
		})
	})
}

// SpanError marks the span with the provided err.
// Returns the same error provided.
func SpanError(span trace.Span, err error) error {
	if err == nil {
		return nil
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, strings.ReplaceAll(err.Error(), paths.UserHome, redactedUserHome))
	sentry.CaptureException(err)
	return err
}

// CaptureError retrieves the span from the ctx and marks it with the provided err.
// Returns the same error provided.
func CaptureError(ctx context.Context, err error) error {
	span := trace.SpanFromContext(ctx)
	return SpanError(span, err)
}

type Shutdown func()

// Init initializes the otel framework.
func Init(ctx context.Context) ([]Shutdown, error) {
	dsn := "https://9e0748223d5bc43e873f811a849e982e@o1009025.ingest.us.sentry.io/4507177762357248"
	// TODO: combine telemetry and trace packages?
	if telemetry.DNT() {
		pterm.Debug.Println("Tracing is disabled")
		dsn = ""
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:                dsn,
		EnableTracing:      true,
		Release:            build.Version,
		TracesSampleRate:   1.0,
		ProfilesSampleRate: 1.0,
		// ServerName can be considered PII, hardcode to N/A
		ServerName:            "N/A",
		BeforeSend:            removePII,
		BeforeSendTransaction: removePII,
	})

	if err != nil {
		return nil, fmt.Errorf("unable to initialize sentry: %w", err)
	}

	cleanups := []Shutdown{func() { sentry.Flush(2 * time.Second) }}

	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			attribute.String("version", build.Version),
		),
	)

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()),
		sdktrace.WithResource(r),
	)
	cleanups = append(cleanups, func() { tracerProvider.Shutdown(ctx) })

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(sentryotel.NewSentryPropagator())

	return cleanups, nil
}

// removePII removes potentially PII information that may be contained within the trace data.
func removePII(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	// message
	event.Message = strings.ReplaceAll(event.Message, paths.UserHome, redactedUserHome)

	// errors
	for _, ex := range event.Exception {
		ex.Value = strings.ReplaceAll(ex.Value, paths.UserHome, redactedUserHome)
	}

	// spans
	for _, span := range event.Spans {
		span.Name = strings.ReplaceAll(span.Name, paths.UserHome, redactedUserHome)
		span.Description = strings.ReplaceAll(span.Description, paths.UserHome, redactedUserHome)
	}

	return event
}
