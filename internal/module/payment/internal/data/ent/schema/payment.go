package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Payment is one payment.
type Payment struct {
	ent.Schema
}

func (Payment) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}),
		field.Int64("amount"),
		field.String("currency"),
		field.String("status").Default("pending"),
	}
}
