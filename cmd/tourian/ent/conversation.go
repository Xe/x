// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"within.website/x/cmd/tourian/ent/conversation"
)

// Conversation is the model entity for the Conversation schema.
type Conversation struct {
	config `json:"-"`
	// ID of the ent.
	ID string `json:"id,omitempty"`
	// PageURL holds the value of the "page_url" field.
	PageURL      string `json:"page_url,omitempty"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Conversation) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case conversation.FieldID, conversation.FieldPageURL:
			values[i] = new(sql.NullString)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Conversation fields.
func (c *Conversation) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case conversation.FieldID:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field id", values[i])
			} else if value.Valid {
				c.ID = value.String
			}
		case conversation.FieldPageURL:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field page_url", values[i])
			} else if value.Valid {
				c.PageURL = value.String
			}
		default:
			c.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Conversation.
// This includes values selected through modifiers, order, etc.
func (c *Conversation) Value(name string) (ent.Value, error) {
	return c.selectValues.Get(name)
}

// Update returns a builder for updating this Conversation.
// Note that you need to call Conversation.Unwrap() before calling this method if this Conversation
// was returned from a transaction, and the transaction was committed or rolled back.
func (c *Conversation) Update() *ConversationUpdateOne {
	return NewConversationClient(c.config).UpdateOne(c)
}

// Unwrap unwraps the Conversation entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (c *Conversation) Unwrap() *Conversation {
	_tx, ok := c.config.driver.(*txDriver)
	if !ok {
		panic("ent: Conversation is not a transactional entity")
	}
	c.config.driver = _tx.drv
	return c
}

// String implements the fmt.Stringer.
func (c *Conversation) String() string {
	var builder strings.Builder
	builder.WriteString("Conversation(")
	builder.WriteString(fmt.Sprintf("id=%v, ", c.ID))
	builder.WriteString("page_url=")
	builder.WriteString(c.PageURL)
	builder.WriteByte(')')
	return builder.String()
}

// Conversations is a parsable slice of Conversation.
type Conversations []*Conversation
