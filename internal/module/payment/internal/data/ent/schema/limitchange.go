package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// LimitChange is this module's record of a SettingChanged event it received.
type LimitChange struct {
	ent.Schema
}

func (LimitChange) Fields() []ent.Field {
	return []ent.Field{
		field.String("currency"),
		field.Int64("max_amount"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}
