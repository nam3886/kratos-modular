package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Setting holds the payment limit of one currency. The ID is the currency code.
type Setting struct {
	ent.Schema
}

func (Setting) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").StorageKey("currency"),
		field.Int64("max_amount"),
	}
}
