package mq_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	settingsv1 "github.com/nam3886/kratos-modular/api/settings/v1"
	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/platform/mq"
	"github.com/nam3886/kratos-modular/internal/platform/testkit"
)

// The handler of an event runs in the trace of the request that published it.
func TestTraceCrossesTheOutbox(t *testing.T) {
	otel.SetTracerProvider(sdktrace.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.TraceContext{})

	dbs := testkit.Postgres(t)
	m, cleanup, err := mq.New(&conf.Data{Rabbitmq: &conf.Data_RabbitMQ{Url: testkit.RabbitMQURL(t)}}, dbs, slog.New(slog.DiscardHandler), prometheus.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	event := &settingsv1.SettingChanged{Currency: "USD", MaxAmount: 1}
	handled := make(chan trace.TraceID, 1)
	if err := m.Handle("trace-"+watermill.NewShortUUID(), mq.Topic(event), func(ctx context.Context, _ *message.Message) error {
		handled <- trace.SpanContextFromContext(ctx).TraceID()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	go func() { _ = m.Start(context.Background()) }()
	defer func() { _ = m.Stop(context.Background()) }()

	ctx, span := otel.Tracer("test").Start(context.Background(), "request")
	defer span.End()
	main, _ := dbs.Get("main")
	tx, err := main.Pool.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := mq.Publish(ctx, tx, watermill.NewUUID(), event); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-handled:
		if want := span.SpanContext().TraceID(); got != want {
			t.Fatalf("handler trace id = %s, want %s", got, want)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("event was not delivered")
	}
}
