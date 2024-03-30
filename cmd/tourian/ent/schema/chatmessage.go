package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// ChatMessage holds the schema definition for the ChatMessage entity.
type ChatMessage struct {
	ent.Schema
}

// Fields of the ChatMessage.
func (ChatMessage) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").NotEmpty(),
		field.String("conversation_id").NotEmpty(),
		field.String("role").NotEmpty(),
		field.String("content").NotEmpty(),
	}
}

// Edges of the ChatMessage.
func (ChatMessage) Edges() []ent.Edge {
	return nil
}
