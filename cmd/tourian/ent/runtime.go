// Code generated by ent, DO NOT EDIT.

package ent

import (
	"within.website/x/cmd/tourian/ent/chatmessage"
	"within.website/x/cmd/tourian/ent/schema"
)

// The init function reads all schema descriptors with runtime code
// (default values, validators, hooks and policies) and stitches it
// to their package variables.
func init() {
	chatmessageFields := schema.ChatMessage{}.Fields()
	_ = chatmessageFields
	// chatmessageDescConversationID is the schema descriptor for conversation_id field.
	chatmessageDescConversationID := chatmessageFields[1].Descriptor()
	// chatmessage.ConversationIDValidator is a validator for the "conversation_id" field. It is called by the builders before save.
	chatmessage.ConversationIDValidator = chatmessageDescConversationID.Validators[0].(func(string) error)
	// chatmessageDescRole is the schema descriptor for role field.
	chatmessageDescRole := chatmessageFields[2].Descriptor()
	// chatmessage.RoleValidator is a validator for the "role" field. It is called by the builders before save.
	chatmessage.RoleValidator = chatmessageDescRole.Validators[0].(func(string) error)
	// chatmessageDescContent is the schema descriptor for content field.
	chatmessageDescContent := chatmessageFields[3].Descriptor()
	// chatmessage.ContentValidator is a validator for the "content" field. It is called by the builders before save.
	chatmessage.ContentValidator = chatmessageDescContent.Validators[0].(func(string) error)
	// chatmessageDescID is the schema descriptor for id field.
	chatmessageDescID := chatmessageFields[0].Descriptor()
	// chatmessage.IDValidator is a validator for the "id" field. It is called by the builders before save.
	chatmessage.IDValidator = chatmessageDescID.Validators[0].(func(string) error)
}