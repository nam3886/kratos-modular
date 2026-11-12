// Package mq is the messaging layer: a transactional outbox in PostgreSQL,
// a forwarder that relays committed messages to RabbitMQ, and consumers with
// retry, a poison queue and an inbox for de-duplication. It is built on
// Watermill.
package mq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	amqp "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	wsql "github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"

	"github.com/nam3886/kratos-modular/internal/conf"
	"github.com/nam3886/kratos-modular/internal/platform/db"
)

const (
	outboxTopic = "outbox"
	// PoisonTopic receives messages a consumer failed to handle after all retries.
	PoisonTopic = "poison"
)

// Handler handles one message. Returning an error triggers a retry.
type Handler func(ctx context.Context, msg *message.Message) error

// MQ runs the forwarders and the consumers. It implements Kratos'
// transport.Server so the application starts and stops it.
type MQ struct {
	url        string
	logger     watermill.LoggerAdapter
	publisher  *amqp.Publisher
	router     *message.Router
	forwarders []*forwarder.Forwarder
	subs       []*amqp.Subscriber

	mu     sync.Mutex // guards cancel: Stop may run concurrently with Start
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New connects to RabbitMQ and prepares one forwarder per database. The
// outbox and inbox tables are created by the migrate command.
func New(c *conf.Data, dbs *db.Manager, log *slog.Logger) (*MQ, func(), error) {
	logger := watermill.NewSlogLogger(log)
	url := c.GetRabbitmq().GetUrl()
	pub, err := amqp.NewPublisher(config(url, "publisher"), logger)
	if err != nil {
		return nil, nil, fmt.Errorf("rabbitmq publisher: %w", err)
	}
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, nil, err
	}
	poison, err := middleware.PoisonQueue(pub, PoisonTopic)
	if err != nil {
		return nil, nil, err
	}
	router.AddMiddleware(
		poison,
		middleware.Retry{MaxRetries: 3, InitialInterval: 100 * time.Millisecond, Multiplier: 2, Logger: logger}.Middleware,
		middleware.Recoverer,
	)
	m := &MQ{url: url, logger: logger, publisher: pub, router: router}
	for _, d := range dbs.All() {
		sub, err := wsql.NewSubscriber(wsql.BeginnerFromStdSQL(d.Pool), wsql.SubscriberConfig{
			SchemaAdapter:  outboxSchema(),
			OffsetsAdapter: wsql.PostgreSQLQueueOffsetsAdapter{DeleteOnAck: true},
			PollInterval:   200 * time.Millisecond,
		}, logger)
		if err != nil {
			return nil, nil, err
		}
		fwd, err := forwarder.NewForwarder(sub, pub, logger, forwarder.Config{ForwarderTopic: outboxTopic})
		if err != nil {
			return nil, nil, err
		}
		m.forwarders = append(m.forwarders, fwd)
	}
	cleanup := func() { _ = pub.Close() }
	return m, cleanup, nil
}

// Handle registers a consumer. Its queue is declared immediately, so events
// published from now on are kept even before the consumer starts.
func (m *MQ) Handle(consumer, topic string, h Handler) error {
	sub, err := amqp.NewSubscriber(config(m.url, consumer), m.logger)
	if err != nil {
		return err
	}
	if err := sub.SubscribeInitialize(topic); err != nil {
		return fmt.Errorf("declare queue for %s on %s: %w", consumer, topic, err)
	}
	m.subs = append(m.subs, sub)
	m.router.AddConsumerHandler(consumer+":"+topic, topic, sub, func(msg *message.Message) error {
		return h(msg.Context(), msg)
	})
	return nil
}

func (m *MQ) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	m.mu.Lock()
	m.cancel = cancel
	m.mu.Unlock()
	errc := make(chan error, len(m.forwarders)+1)
	for _, f := range m.forwarders {
		m.wg.Go(func() { errc <- f.Run(ctx) })
	}
	m.wg.Go(func() { errc <- m.router.Run(ctx) })
	go func() { m.wg.Wait(); close(errc) }()
	var errs []error
	for err := range errc {
		if err != nil && !errors.Is(err, context.Canceled) {
			errs = append(errs, err)
			cancel()
		}
	}
	return errors.Join(errs...)
}

// Stop lets in-flight messages finish and stops forwarding and consuming.
func (m *MQ) Stop(context.Context) error {
	m.mu.Lock()
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.Unlock()
	var errs []error
	for _, f := range m.forwarders {
		errs = append(errs, f.Close())
	}
	errs = append(errs, m.router.Close())
	m.wg.Wait()
	for _, s := range m.subs {
		errs = append(errs, s.Close())
	}
	return errors.Join(errs...)
}

// config is a durable fan-out exchange per topic and one durable queue per
// consumer; publishes wait for the broker's confirmation.
func config(url, consumer string) amqp.Config {
	cfg := amqp.NewDurablePubSubConfig(url, amqp.GenerateQueueNameTopicNameWithSuffix(consumer))
	cfg.Publish.ConfirmDelivery = true
	return cfg
}
