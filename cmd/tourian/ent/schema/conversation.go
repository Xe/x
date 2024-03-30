package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Conversation holds the schema definition for the Conversation entity.
type Conversation struct {
	ent.Schema
}

// Fields of the Conversation.
func (Conversation) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty(),
		field.String("page_url").NotEmpty(),
	}
}

// Edges of the Conversation.
func (Conversation) Edges() []ent.Edge {
	return nil
}
