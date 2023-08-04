package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	irc "github.com/thoj/go-ircevent"
	"go.jetpack.io/tyson"
	"golang.org/x/exp/slog"
	"honnef.co/go/transmission"
	"tailscale.com/hostinfo"
	"tailscale.com/jsondb"
	"within.website/x/internal"
	"within.website/x/web/parsetorrentname"
)

var (
	dbLoc       = flag.String("db-loc", "./data.json", "path to data file")
	tysonConfig = flag.String("tyson-config", "./config.ts", "path to configuration secrets (TySON)")

	annRegex = regexp.MustCompile(`^New Torrent Announcement: <([^>]*)>\s+Name:'(.*)' uploaded by '.*' ?(freeleech)?\s+-\s+https://\w+.\w+.\w+./\w+./([0-9]+)$`)
)

func ConvertURL(torrentID, rssKey, name string) string {
	name = strings.ReplaceAll(name, " ", ".") + ".torrent"
	return fmt.Sprintf("https://www.torrentleech.org/rss/download/%s/%s/%s", torrentID, rssKey, name)
}

type TorrentAnnouncement struct {
	Category  string
	Name      string
	Freeleech bool
	TorrentID string
}

func ParseTorrentAnnouncement(input string) (*TorrentAnnouncement, error) {
	match := annRegex.FindStringSubmatch(input)

	if match == nil {
		return nil, fmt.Errorf("invalid torrent announcement format")
	}

	torrent := &TorrentAnnouncement{
		Category:  match[1],
		Name:      strings.TrimSpace(match[2]),
		Freeleech: match[3] != "",
		TorrentID: match[4],
	}

	return torrent, nil
}

func main() {
	internal.HandleStartup()
	hostinfo.SetApp("within.website/x/cmd/sanguisuga")

	var c Config
	if err := tyson.Unmarshal(*tysonConfig, &c); err != nil {
		log.Fatalf("can't unmarshal config: %v", err)
	}

	db, err := jsondb.Open[State](*dbLoc)
	if err != nil {
		log.Fatalf("can't set up database: %v", err)
	}
	if db.Data == nil {
		db.Data = &State{
			Seen: map[string]TorrentAnnouncement{},
		}
	}
	if err := db.Save(); err != nil {
		log.Fatalf("can't ping database: %v", err)
	}

	cl := &transmission.Client{
		Client:   http.DefaultClient,
		Endpoint: c.Transmission.URL,
		Username: c.Transmission.User,
		Password: c.Transmission.Password,
	}

	s := &Sanguisuga{
		Config: c,
		cl:     cl,
		db:     db,
	}

	ircCli := irc.IRC(c.IRC.Nick, c.IRC.User)
	ircCli.Password = c.IRC.Password
	ircCli.RealName = c.IRC.Real
	ircCli.AddCallback("PRIVMSG", s.HandleIRCMessage)
	ircCli.AddCallback("001", func(ev *irc.Event) {
		ircCli.Join(c.IRC.Channel)
	})
	ircCli.Log = slog.NewLogLogger(slog.Default().Handler().WithAttrs([]slog.Attr{slog.String("from", "ircevent")}), slog.LevelInfo)
	ircCli.Timeout = 5 * time.Second

	if err := ircCli.Connect(c.IRC.Server); err != nil {
		log.Fatalf("can't connect to IRC server %s: %v", c.IRC.Server, err)
	}

	ircCli.Loop()
}

type Sanguisuga struct {
	Config Config
	cl     *transmission.Client
	db     *jsondb.DB[State]
}

type State struct {
	// Name + " " + SeasonEpisode -> TorrentAnnouncement
	Seen map[string]TorrentAnnouncement
}

func (s *Sanguisuga) HandleIRCMessage(ev *irc.Event) {
	// check if in channel
	if ev.Code != "PRIVMSG" {
		return
	}

	if ev.Arguments[0] != s.Config.IRC.Channel {
		return
	}

	ta, err := ParseTorrentAnnouncement(ev.MessageWithoutFormat())
	if err != nil {
		slog.Debug("can't parse torrent announcment", "err", err, "msg", ev.MessageWithoutFormat())
		return
	}

	slog.Debug("found torrent announcment", "category", ta.Category, "freeleech", ta.Freeleech, "name", ta.Name)

	if ta.Category == "TV :: Episodes HD" {
		ti, err := parsetorrentname.Parse(ta.Name)
		if err != nil {
			slog.Debug("can't parse ShowMeta", "err", err, "name", ta.Name)
			return
		}
		id := fmt.Sprintf("S%02dE%02d", ti.Season, ti.Episode)
		slog.Debug("found ShowMeta", "title", ti.Title, "id", id, "quality", ti.Resolution, "group", ti.Group)

		stateKey := fmt.Sprintf("%s %s", ti.Title, id)

		for _, show := range s.Config.Shows {
			if s.db.Data == nil {
				s.db.Data = &State{
					Seen: map[string]TorrentAnnouncement{},
				}
			}
			if _, found := s.db.Data.Seen[stateKey]; found {
				slog.Info("already snatched", "title", ti.Title, "id", id)
				return
			}

			if show.Title != ti.Title {
				slog.Debug("wrong name")
				continue
			}

			if show.Quality != ti.Resolution {
				slog.Debug("wrong resolution")
				continue
			}

			torrentURL := ConvertURL(ta.TorrentID, s.Config.RSSKey, ta.Name)

			slog.Debug("found url", "url", torrentURL)
			downloadDir := filepath.Join(show.DiskPath, fmt.Sprintf("Season %02d", ti.Season))

			var buf bytes.Buffer
			resp, err := http.Get(torrentURL)
			if err != nil {
				slog.Error("can't download torrent", "url", torrentURL, "err", err, "torrentID", ta.TorrentID)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				slog.Error("got wrong status code", "want", http.StatusOK, "got", resp.StatusCode, "url", torrentURL, "torrentID", ta.TorrentID)
				continue
			}

			defer resp.Body.Close()
			if n, err := io.Copy(&buf, resp.Body); err != nil {
				slog.Error("can't fetch torrent body", "url", torrentURL, "err", err, "torrentID", ta.TorrentID)
				continue
			} else {
				slog.Info("downloaded bytes", "n", n, "url", torrentURL)
			}

			metaInfo := base64.StdEncoding.EncodeToString(buf.Bytes())

			t, dupe, err := s.cl.AddTorrent(&transmission.NewTorrent{
				DownloadDir: downloadDir,
				Metainfo:    metaInfo,
				Paused:      false,
			})
			if err != nil {
				slog.Error("error adding torrent", "url", torrentURL, "err", err, "torrentID", ta.TorrentID)
				return
			}

			slog.Info("added torrent", "title", ti.Title, "id", id, "path", downloadDir, "infohash", t.Hash, "tid", t.ID, "dupe", dupe)

			s.db.Data.Seen[stateKey] = *ta
			if err := s.db.Save(); err != nil {
				slog.Error("error saving state", "err", err)
			}
		}
	}
}
