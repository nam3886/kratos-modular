package mq_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	amqp "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/platform/mq"
	"github.com/nam3886/kratos-modular/internal/platform/testkit"
)

// A message interrupted by shutdown must be redelivered after restart.
func TestStopRedeliversInFlightMessage(t *testing.T) {
	url := testkit.RabbitMQURL(t)
	dbs := testkit.Postgres(t)
	newMQ := func() *mq.MQ {
		m, cleanup, err := mq.New(&conf.Data{Rabbitmq: &conf.Data_RabbitMQ{Url: url}}, dbs, slog.New(slog.DiscardHandler), prometheus.NewRegistry())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(cleanup)
		return m
	}
	topic := "test.shutdown." + watermill.NewShortUUID()

	m := newMQ()
	started := make(chan struct{})
	if err := m.Handle("c", topic, func(ctx context.Context, _ *message.Message) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}

	pub, err := amqp.NewPublisher(amqp.NewDurablePubSubConfig(url, nil), watermill.NopLogger{})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pub.Close() }()
	if err := pub.Publish(topic, message.NewMessage(watermill.NewUUID(), nil)); err != nil {
		t.Fatal(err)
	}

	go func() { _ = m.Start(context.Background()) }()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("handler never started")
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	m2 := newMQ()
	redelivered := make(chan struct{})
	if err := m2.Handle("c", topic, func(context.Context, *message.Message) error {
		close(redelivered)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	go func() { _ = m2.Start(context.Background()) }()
	defer func() { _ = m2.Stop(context.Background()) }()
	select {
	case <-redelivered:
	case <-time.After(10 * time.Second):
		t.Fatal("message was not redelivered")
	}
}
