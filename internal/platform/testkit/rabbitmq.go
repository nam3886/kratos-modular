package testkit

import (
	"context"
	"sync"
	"testing"

	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

var (
	mqOnce sync.Once
	mqURL  string
	mqErr  error
)

func RabbitMQURL(t testing.TB) string {
	t.Helper()
	mqOnce.Do(func() {
		ctx := context.Background()
		ctr, err := rabbitmq.Run(ctx, "rabbitmq:4-alpine")
		if err != nil {
			mqErr = err
			return
		}
		mqURL, mqErr = ctr.AmqpURL(ctx)
	})
	if mqErr != nil {
		t.Fatalf("start rabbitmq: %v", mqErr)
	}
	return mqURL
}
