package slog

import (
	"context"
	"log/slog"
	"slices"
)

// ctxAttrsKey is the private context key under which ContextWithAttrs stores
// accumulated attributes.
type ctxAttrsKey struct{}

// ContextWithAttrs returns a copy of ctx carrying attrs, in addition to any
// attributes already attached to ctx by an outer ContextWithAttrs call. A
// [ContextHandler] emits these attributes on every record logged with the
// returned context.
//
// Calls accumulate: attrs from an outer call and attrs from an inner call are
// all emitted. Each call allocates a fresh, exactly-sized slice, so deriving
// two children from the same parent context never lets one child's attributes
// leak into the other.
func ContextWithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	prev := attrsFromContext(ctx)
	merged := make([]slog.Attr, 0, len(prev)+len(attrs))
	merged = append(merged, prev...)
	merged = append(merged, attrs...)
	return context.WithValue(ctx, ctxAttrsKey{}, merged)
}

// attrsFromContext returns the attributes attached to ctx, or nil.
func attrsFromContext(ctx context.Context) []slog.Attr {
	attrs, _ := ctx.Value(ctxAttrsKey{}).([]slog.Attr)
	return attrs
}

// handlerOp is a single recorded WithAttrs or WithGroup operation. Exactly one
// of group / attrs is meaningful: a non-empty group means WithGroup, otherwise
// it is a WithAttrs carrying attrs.
type handlerOp struct {
	group string
	attrs []slog.Attr
}

// ContextHandler is a [slog.Handler] that emits the attributes attached to a
// record's context by [ContextWithAttrs]. The context attributes are always
// written at the top level of the record, never nested inside a group opened
// with WithGroup, so that call-scoped fields (request IDs, user IDs, ...) stay
// queryable regardless of how a derived logger is grouped.
type ContextHandler struct {
	// base is the wrapped handler before any WithAttrs / WithGroup applied to
	// this ContextHandler. Context attributes are injected here, ahead of ops,
	// so they land above any group.
	base slog.Handler
	// ops records the WithAttrs / WithGroup calls made on this handler, in
	// order, to be replayed after the context attributes are injected.
	ops []handlerOp
	// inner is base with ops already applied. It handles records that carry no
	// context attributes, and answers Enabled, without per-record rebuilding.
	inner slog.Handler
	// grouped reports whether any op opens a group. When false, context
	// attributes can be appended straight to the record cheaply.
	grouped bool
}

// NewContextHandler wraps inner so that attributes attached with
// [ContextWithAttrs] are emitted on every record. The returned handler
// delegates Enabled, WithAttrs, and WithGroup to inner.
func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{base: inner, inner: inner}
}

// Enabled delegates to the wrapped handler.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle injects the context attributes at the top level of rec and delegates
// to the wrapped handler.
func (h *ContextHandler) Handle(ctx context.Context, rec slog.Record) error {
	attrs := attrsFromContext(ctx)
	if len(attrs) == 0 {
		return h.inner.Handle(ctx, rec)
	}

	// No group is open, so appending to the record keeps the attributes at the
	// top level. Clone first: the caller still owns rec.
	if !h.grouped {
		rec = rec.Clone()
		rec.AddAttrs(attrs...)
		return h.inner.Handle(ctx, rec)
	}

	// A group is open. Adding the attributes to the record would bury them
	// inside that group, so rebuild the handler chain with the context
	// attributes applied to base, ahead of the recorded groups.
	hdl := h.base.WithAttrs(slices.Clone(attrs))
	for _, op := range h.ops {
		if op.group != "" {
			hdl = hdl.WithGroup(op.group)
		} else {
			hdl = hdl.WithAttrs(op.attrs)
		}
	}
	return hdl.Handle(ctx, rec)
}

// WithAttrs delegates to the wrapped handler and re-wraps the result so the
// context behaviour survives on the derived handler.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return &ContextHandler{
		base:    h.base,
		ops:     append(slices.Clip(h.ops), handlerOp{attrs: attrs}),
		inner:   h.inner.WithAttrs(attrs),
		grouped: h.grouped,
	}
}

// WithGroup delegates to the wrapped handler and re-wraps the result so the
// context behaviour survives on the derived handler. Context attributes
// continue to be emitted above the group.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &ContextHandler{
		base:    h.base,
		ops:     append(slices.Clip(h.ops), handlerOp{group: name}),
		inner:   h.inner.WithGroup(name),
		grouped: true,
	}
}
