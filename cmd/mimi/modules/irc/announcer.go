package irc

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"
	"tailscale.com/jsondb"
	"within.website/x/cmd/mimi/internal"
	"within.website/x/proto/external/jsonfeed"
	"within.website/x/proto/mimi/announce"
)

func (m *Module) RegisterHTTP(mux *http.ServeMux) {
	mux.Handle(announce.AnnouncePathPrefix, announce.NewAnnounceServer(&AnnounceService{Module: m}))
}

type AnnounceService struct {
	*Module
	db   *jsondb.DB[State]
	lock sync.Mutex
}

type State struct {
	Announced map[string]struct{}
}

func (a *AnnounceService) initDB() error {
	if a.db != nil {
		slog.Debug("db already initialized")
		return nil
	}

	fname := filepath.Join(internal.DataDir(), "announced.json")
	slog.Debug("opening jsondb for announced items", "fname", fname)

	if _, err := os.Stat(fname); os.IsNotExist(err) {
		os.WriteFile(fname, []byte(`{"Announced":{}}`), 0644)
	}

	db, err := jsondb.Open[State](fname)
	if err != nil {
		return err
	}
	db.Save()

	if db.Data == nil {
		slog.Debug("creating empty state")
		db.Data = &State{
			Announced: map[string]struct{}{
				"https://within.website/": struct{}{},
			},
		}
		if err := db.Save(); err != nil {
			return err
		}
	}

	a.db = db

	return nil
}

func (a *AnnounceService) Announce(ctx context.Context, msg *jsonfeed.Item) (*emptypb.Empty, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if err := a.initDB(); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	if msg.Title == "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "missing title")
	}
	if msg.Url == "" {
		return nil, twirp.NewError(twirp.InvalidArgument, "missing url")
	}

	if _, ok := a.db.Data.Announced[msg.Url]; ok {
		return &emptypb.Empty{}, nil
	}

	a.db.Data.Announced[msg.Url] = struct{}{}
	if err := a.db.Save(); err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	a.conn.Privmsgf(*ircChannel, "New article :: %s :: %s", msg.Title, msg.Url)

	return &emptypb.Empty{}, nil
}
