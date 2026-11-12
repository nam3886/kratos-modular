package server

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/nam3886/kratos-modular/internal/conf"
)

// SetupTracing installs the global tracer provider. Every request then has a
// trace id (visible in logs); spans are exported only when an endpoint is set.
func SetupTracing(ctx context.Context, c *conf.Trace, name, version string) (func(), error) {
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(resource.NewSchemaless(semconv.ServiceName(name), semconv.ServiceVersion(version))),
	}
	if c.GetEndpoint() != "" {
		exp, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(c.GetEndpoint()), otlptracegrpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
	}, nil
}
