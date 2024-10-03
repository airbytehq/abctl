package trace

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/airbytehq/abctl/internal/build"
	"github.com/getsentry/sentry-go"
	sentryotel "github.com/getsentry/sentry-go/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/airbytehq/abctl/trace"

var (
	// may not be required
	once   sync.Once
	tracer trace.Tracer
)

func NewSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	once.Do(func() {
		tracer = otel.Tracer(tracerName)
	})
	return tracer.Start(ctx, name)
}

func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

func SpanError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func CaptureError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.AddAttachment(&sentry.Attachment{
			Filename:    "test.log",
			ContentType: "text/plain",
			Payload:     []byte("this is a test log.\nonly the best log.\nnever the rest log."),
		})
		sentry.CaptureException(err)
	})
}

type Shutdown func()

func Init(ctx context.Context) ([]Shutdown, error) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:                "https://9e0748223d5bc43e873f811a849e982e@o1009025.ingest.us.sentry.io/4507177762357248",
		EnableTracing:      true,
		Debug:              true,
		Environment:        "dev",
		Release:            build.Version,
		TracesSampleRate:   1.0,
		ProfilesSampleRate: 1.0,
		// ServerName can be considered PII, hardcode to N/A
		ServerName: "N/A",
		//BeforeSend:            eventRemovePII,
		//BeforeSendTransaction: txRemovePII,
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

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()),
		sdktrace.WithResource(r),
	)
	cleanups = append(cleanups, func() { tp.Shutdown(ctx) })

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(sentryotel.NewSentryPropagator())

	return cleanups, nil
}

func eventRemovePII(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	return nil
}

func txRemovePII(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	return nil
}
