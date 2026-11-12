package mq

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	wsql "github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/proto"
)

// Execer is a running transaction: *sql.Tx or a generated *ent.Tx.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type executor struct{ tx Execer }

func (e executor) ExecContext(ctx context.Context, q string, args ...any) (wsql.Result, error) {
	return e.tx.ExecContext(ctx, q, args...)
}

func (e executor) QueryContext(ctx context.Context, q string, args ...any) (wsql.Rows, error) {
	return e.tx.QueryContext(ctx, q, args...)
}

var outboxSchema = wsql.PostgreSQLQueueSchema{}

// Publish stores event in the outbox through tx. The event reaches RabbitMQ
// only if tx commits. The topic is the event's full proto name.
func Publish(ctx context.Context, tx Execer, id string, event proto.Message) error {
	payload, err := proto.Marshal(event)
	if err != nil {
		return err
	}
	sqlPub, err := wsql.NewPublisher(executor{tx}, wsql.PublisherConfig{SchemaAdapter: outboxSchema}, watermill.NopLogger{})
	if err != nil {
		return err
	}
	pub := forwarder.NewPublisher(sqlPub, forwarder.PublisherConfig{ForwarderTopic: outboxTopic})
	msg := message.NewMessage(id, payload)
	msg.SetContext(ctx)
	return pub.Publish(Topic(event), msg)
}

func Topic(event proto.Message) string {
	return string(event.ProtoReflect().Descriptor().FullName())
}

func InitTables(ctx context.Context, pool *sql.DB) error {
	queries, err := outboxSchema.SchemaInitializingQueries(wsql.SchemaInitializingQueriesParams{Topic: outboxTopic})
	if err != nil {
		return err
	}
	for _, q := range queries {
		if _, err := pool.ExecContext(ctx, q.Query, q.Args...); err != nil {
			return fmt.Errorf("create outbox table: %w", err)
		}
	}
	_, err = pool.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS inbox (
		consumer   TEXT NOT NULL,
		message_id TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		PRIMARY KEY (consumer, message_id)
	)`)
	if err != nil {
		return fmt.Errorf("create inbox table: %w", err)
	}
	return nil
}

// Pending counts the events waiting in the outbox of one database.
func Pending(pool *sql.DB) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var n float64
	if err := pool.QueryRowContext(ctx, `SELECT count(*) FROM "watermill_`+outboxTopic+`"`).Scan(&n); err != nil {
		return -1
	}
	return n
}
