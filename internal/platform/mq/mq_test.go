package mq_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	amqp "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill/message"
	amqp091 "github.com/rabbitmq/amqp091-go"

	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/platform/mq"
	"github.com/nam3886/kratos-modular/internal/platform/testkit"
)

// A message interrupted by shutdown must be redelivered after restart, not
// moved to the poison queue.
func TestStopDoesNotPoisonInFlightMessage(t *testing.T) {
	url := testkit.RabbitMQURL(t)
	m, cleanup, err := mq.New(&conf.Data{Rabbitmq: &conf.Data_RabbitMQ{Url: url}}, testkit.Postgres(t), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	topic := "test.shutdown." + watermill.NewShortUUID()
	started := make(chan struct{})
	if err := m.Handle("c", topic, func(ctx context.Context, _ *message.Message) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}); err != nil {
		t.Fatal(err)
	}

	conn, err := amqp091.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	ready := func(queue string) int {
		q, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil)
		if err != nil {
			t.Fatalf("queue %s: %v", queue, err)
		}
		return q.Messages
	}
	poisoned := ready("poison_" + mq.PoisonTopic)

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

	deadline := time.Now().Add(5 * time.Second)
	for ready(topic+"_c") != 1 {
		if time.Now().After(deadline) {
			t.Fatal("message was not returned to its queue")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if got := ready("poison_" + mq.PoisonTopic); got != poisoned {
		t.Fatalf("poison queue grew from %d to %d", poisoned, got)
	}
}
