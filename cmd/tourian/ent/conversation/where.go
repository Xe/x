// Code generated by ent, DO NOT EDIT.

package conversation

import (
	"entgo.io/ent/dialect/sql"
	"within.website/x/cmd/tourian/ent/predicate"
)

// ID filters vertices based on their ID field.
func ID(id string) predicate.Conversation {
	return predicate.Conversation(sql.FieldEQ(FieldID, id))
}

// IDEQ applies the EQ predicate on the ID field.
func IDEQ(id string) predicate.Conversation {
	return predicate.Conversation(sql.FieldEQ(FieldID, id))
}

// IDNEQ applies the NEQ predicate on the ID field.
func IDNEQ(id string) predicate.Conversation {
	return predicate.Conversation(sql.FieldNEQ(FieldID, id))
}

// IDIn applies the In predicate on the ID field.
func IDIn(ids ...string) predicate.Conversation {
	return predicate.Conversation(sql.FieldIn(FieldID, ids...))
}

// IDNotIn applies the NotIn predicate on the ID field.
func IDNotIn(ids ...string) predicate.Conversation {
	return predicate.Conversation(sql.FieldNotIn(FieldID, ids...))
}

// IDGT applies the GT predicate on the ID field.
func IDGT(id string) predicate.Conversation {
	return predicate.Conversation(sql.FieldGT(FieldID, id))
}

// IDGTE applies the GTE predicate on the ID field.
func IDGTE(id string) predicate.Conversation {
	return predicate.Conversation(sql.FieldGTE(FieldID, id))
}

// IDLT applies the LT predicate on the ID field.
func IDLT(id string) predicate.Conversation {
	return predicate.Conversation(sql.FieldLT(FieldID, id))
}

// IDLTE applies the LTE predicate on the ID field.
func IDLTE(id string) predicate.Conversation {
	return predicate.Conversation(sql.FieldLTE(FieldID, id))
}

// IDEqualFold applies the EqualFold predicate on the ID field.
func IDEqualFold(id string) predicate.Conversation {
	return predicate.Conversation(sql.FieldEqualFold(FieldID, id))
}

// IDContainsFold applies the ContainsFold predicate on the ID field.
func IDContainsFold(id string) predicate.Conversation {
	return predicate.Conversation(sql.FieldContainsFold(FieldID, id))
}

// PageURL applies equality check predicate on the "page_url" field. It's identical to PageURLEQ.
func PageURL(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldEQ(FieldPageURL, v))
}

// PageURLEQ applies the EQ predicate on the "page_url" field.
func PageURLEQ(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldEQ(FieldPageURL, v))
}

// PageURLNEQ applies the NEQ predicate on the "page_url" field.
func PageURLNEQ(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldNEQ(FieldPageURL, v))
}

// PageURLIn applies the In predicate on the "page_url" field.
func PageURLIn(vs ...string) predicate.Conversation {
	return predicate.Conversation(sql.FieldIn(FieldPageURL, vs...))
}

// PageURLNotIn applies the NotIn predicate on the "page_url" field.
func PageURLNotIn(vs ...string) predicate.Conversation {
	return predicate.Conversation(sql.FieldNotIn(FieldPageURL, vs...))
}

// PageURLGT applies the GT predicate on the "page_url" field.
func PageURLGT(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldGT(FieldPageURL, v))
}

// PageURLGTE applies the GTE predicate on the "page_url" field.
func PageURLGTE(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldGTE(FieldPageURL, v))
}

// PageURLLT applies the LT predicate on the "page_url" field.
func PageURLLT(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldLT(FieldPageURL, v))
}

// PageURLLTE applies the LTE predicate on the "page_url" field.
func PageURLLTE(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldLTE(FieldPageURL, v))
}

// PageURLContains applies the Contains predicate on the "page_url" field.
func PageURLContains(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldContains(FieldPageURL, v))
}

// PageURLHasPrefix applies the HasPrefix predicate on the "page_url" field.
func PageURLHasPrefix(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldHasPrefix(FieldPageURL, v))
}

// PageURLHasSuffix applies the HasSuffix predicate on the "page_url" field.
func PageURLHasSuffix(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldHasSuffix(FieldPageURL, v))
}

// PageURLEqualFold applies the EqualFold predicate on the "page_url" field.
func PageURLEqualFold(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldEqualFold(FieldPageURL, v))
}

// PageURLContainsFold applies the ContainsFold predicate on the "page_url" field.
func PageURLContainsFold(v string) predicate.Conversation {
	return predicate.Conversation(sql.FieldContainsFold(FieldPageURL, v))
}

// And groups predicates with the AND operator between them.
func And(predicates ...predicate.Conversation) predicate.Conversation {
	return predicate.Conversation(sql.AndPredicates(predicates...))
}

// Or groups predicates with the OR operator between them.
func Or(predicates ...predicate.Conversation) predicate.Conversation {
	return predicate.Conversation(sql.OrPredicates(predicates...))
}

// Not applies the not operator on the given predicate.
func Not(p predicate.Conversation) predicate.Conversation {
	return predicate.Conversation(sql.NotPredicates(p))
}