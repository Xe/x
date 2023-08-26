package resolveHandle

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pasztorpisti/qs"
	"within.website/x/web/bsky"
)

type Query struct {
	Handle *string `json:"handle"`
}

func (q Query) XRPCType() string {
	return "params"
}

type Output struct {
	DID string `json:"did"`
}

func (o Output) XRPCType() string {
	return "object"
}

type Handler interface {
	IdentityResolveHandle(context.Context, *Query) (*Output, error)
}

func ServeHTTP(h Handler) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		q := &Query{}

		if err := qs.Unmarshal(q, req.URL.RawQuery); err != nil {
			slog.Error("error parsing request query parameters", "err", err)
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(rw).Encode(bsky.Error{
				ErrorKind: "InvalidRequestError",
				Message:   "Your request cannot be parsed. Try again.",
			})
			return
		}

		output, err := h.IdentityResolveHandle(req.Context(), q)
		if err != nil {
			slog.Error("error doing handler logic", "err", err)
			switch err.(type) {
			case *bsky.Error:
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(rw).Encode(err)
			default:
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(rw).Encode(bsky.Error{
					ErrorKind: "InternalServerError",
					Message:   "There was an internal server error. No further information is available.",
				})
			}
			return
		}

		json.NewEncoder(rw).Encode(output)
	})
}
