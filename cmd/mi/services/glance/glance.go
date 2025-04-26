package glance

import (
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"within.website/x/cmd/mi/models"
)

//go:generate go tool templ generate

func New(dao *models.DAO) *Glance {
	return &Glance{dao: dao}
}

type Glance struct {
	dao *models.DAO
}

func (g *Glance) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Widget-Title", "Who is front?")
	w.Header().Add("Widget-Content-Type", "html")

	sw, err := g.dao.WhoIsFront(r.Context())
	if err != nil {
		slog.Error("can't query front", "err", err)
		templ.Handler(ohNoes(err), templ.WithStatus(http.StatusInternalServerError)).ServeHTTP(w, r)
		return
	}

	templ.Handler(whoIsFront(sw)).ServeHTTP(w, r)
}
