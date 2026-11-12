package server

import "github.com/nam3886/kratos-modular/internal/platform/mq"

type Messaging struct {
	*mq.MQ
}

// NewMessaging subscribes every consuming module before anything starts.
func NewMessaging(q *mq.MQ, consumers []Consumer) (*Messaging, error) {
	for _, c := range consumers {
		if err := c.RegisterConsumers(q); err != nil {
			return nil, err
		}
	}
	return &Messaging{MQ: q}, nil
}
