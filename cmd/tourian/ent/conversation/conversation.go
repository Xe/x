// Code generated by ent, DO NOT EDIT.

package conversation

import (
	"entgo.io/ent/dialect/sql"
)

const (
	// Label holds the string label denoting the conversation type in the database.
	Label = "conversation"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldPageURL holds the string denoting the page_url field in the database.
	FieldPageURL = "page_url"
	// Table holds the table name of the conversation in the database.
	Table = "conversations"
)

// Columns holds all SQL columns for conversation fields.
var Columns = []string{
	FieldID,
	FieldPageURL,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

var (
	// PageURLValidator is a validator for the "page_url" field. It is called by the builders before save.
	PageURLValidator func(string) error
	// IDValidator is a validator for the "id" field. It is called by the builders before save.
	IDValidator func(string) error
)

// OrderOption defines the ordering options for the Conversation queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByPageURL orders the results by the page_url field.
func ByPageURL(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldPageURL, opts...).ToFunc()
}
