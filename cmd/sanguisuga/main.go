package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hekmon/transmissionrpc/v2"
	irc "github.com/thoj/go-ircevent"
	"go.jetpack.io/tyson"
	"golang.org/x/exp/slog"
	"tailscale.com/hostinfo"
	"tailscale.com/jsondb"
	"within.website/x/internal"
)

var (
	dbLoc       = flag.String("db-loc", "./data.json", "path to data file")
	tysonConfig = flag.String("tyson-config", "./config.ts", "path to configuration secrets (TySON)")
	slogLevel   = flag.String("slog-level", "INFO", "log level")

	annRegex = regexp.MustCompile(`^New Torrent Announcement: <([^>]*)>\s+Name:'(.*)' uploaded by '.*' ?(freeleech)?\s+-\s+https://\w+.\w+.\w+./\w+./([0-9]+)$`)
	showName = regexp.MustCompile(`^(.*)\s+(S[0-9]+E[0-9]+)\s+([0-9]+p)\s+(\w+)\s+(.*)$`)
)

type ShowMeta struct {
	Name          string
	SeasonEpisode *SeasonEpisode
	Quality       string
	Kind          string
	Group         string
}

func (sm ShowMeta) StateKey() string {
	return fmt.Sprintf("%s %s", sm.Name, sm.SeasonEpisode)
}

func ParseShowMeta(input string) (*ShowMeta, error) {
	match := showName.FindStringSubmatch(input)

	if match == nil {
		return nil, fmt.Errorf("invalid input for TV show name: %q", input)
	}

	result := ShowMeta{
		Name:    strings.TrimSpace(match[1]),
		Quality: match[3],
		Kind:    match[4],
		Group:   match[5],
	}

	se, err := ParseSeasonEpisode(match[2])
	if err != nil {
		return nil, err
	}

	result.SeasonEpisode = se

	return &result, nil
}

type SeasonEpisode struct {
	Season  string
	Episode string
}

func (se SeasonEpisode) GetFormattedSeason() string {
	return "Season " + se.Season
}

func (se *SeasonEpisode) String() string {
	return "S" + se.Season + "E" + se.Episode
}

func ParseSeasonEpisode(input string) (*SeasonEpisode, error) {
	re := regexp.MustCompile(`S([0-9]+)E([0-9]+)`)
	match := re.FindStringSubmatch(input)

	if match == nil {
		return nil, fmt.Errorf("invalid input for SeasonEpisode: %q", input)
	}

	season := match[1]
	episode := match[2]
	se := &SeasonEpisode{
		Season:  season,
		Episode: episode,
	}

	return se, nil
}

func ConvertURL(torrentID, rssKey, name string) string {
	name = strings.ReplaceAll(name, " ", ".") + ".torrent"
	return fmt.Sprintf("https://torrentleech.org/rss/download/%s/%s/%s", torrentID, rssKey, name)
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
		slog.Error("can't unmarshal config", "err", err)
		os.Exit(1)
	}

	db, err := jsondb.Open[State](*dbLoc)
	if err != nil {
		slog.Error("can't set up database", "err", err)
		os.Exit(1)
	}
	if db.Data == nil {
		db.Data = &State{
			Seen: map[string]TorrentAnnouncement{},
		}
	}
	if err := db.Save(); err != nil {
		slog.Error("can't ping database", "err", err)
		os.Exit(1)
	}

	tm, err := transmissionrpc.New(c.Transmission.Host, c.Transmission.User, c.Transmission.Password, &transmissionrpc.AdvancedConfig{
		Port:   443,
		HTTPS:  c.Transmission.HTTPS,
		RPCURI: c.Transmission.RPCURI,
	})
	if err != nil {
		slog.Error("can't connect to transmission", "err", err)
		os.Exit(1)
	}
	_ = tm

	portOpen, err := tm.PortTest(context.Background())
	if err != nil {
		slog.Error("can't test if port is open", "err", err)
		os.Exit(1)
	}

	slog.Info("port status", "open", portOpen)

	s := &Sanguisuga{
		Config: c,
		tc:     tm,
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
		slog.Error("can't connect to IRC server", "server", c.IRC.Server, "err", err)
		os.Exit(1)
	}

	ircCli.Loop()
}

type Sanguisuga struct {
	Config Config
	tc     *transmissionrpc.Client
	db     *jsondb.DB[State]
}

func (s *Sanguisuga) DelayedStartTorrent(tid int64) {
	s.tc.TorrentStopIDs(context.Background(), []int64{tid})
	time.Sleep(5 * time.Second) // delay a bit
	s.tc.TorrentStartNowIDs(context.Background(), []int64{tid})
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
		sm, err := ParseShowMeta(ta.Name)
		if err != nil {
			slog.Debug("can't parse ShowMeta", "err", err, "name", ta.Name)
			return
		}
		id := sm.SeasonEpisode.String()
		slog.Debug("found ShowMeta", "title", sm.Name, "id", id, "quality", sm.Quality, "group", sm.Group)

		for _, show := range s.Config.Shows {
			if s.db.Data == nil {
				s.db.Data = &State{
					Seen: map[string]TorrentAnnouncement{},
				}
			}
			if _, found := s.db.Data.Seen[sm.StateKey()]; found {
				slog.Info("already snatched", "title", sm.Name, "id", id)
				return
			}

			if show.Title != sm.Name {
				slog.Debug("wrong name")
				continue
			}

			if show.Quality != sm.Quality {
				slog.Debug("wrong quality")
				continue
			}

			torrentURL := ConvertURL(ta.TorrentID, s.Config.RSSKey, ta.Name)

			slog.Debug("found url", "url", torrentURL)
			downloadDir := filepath.Join(show.DiskPath, sm.SeasonEpisode.GetFormattedSeason())

			t, err := s.tc.TorrentAdd(context.Background(), transmissionrpc.TorrentAddPayload{
				DownloadDir: &downloadDir,
				Filename:    aws.String(torrentURL),
				Paused:      aws.Bool(false),
			})
			if err != nil {
				slog.Error("error adding torrent", "err", err, "torrentID", ta.TorrentID)
				return
			}

			go s.DelayedStartTorrent(*t.ID)

			slog.Info("added torrent", "title", sm.Name, "id", id, "path", downloadDir)

			s.db.Data.Seen[sm.StateKey()] = *ta
			if err := s.db.Save(); err != nil {
				slog.Error("error saving state", "err", err)
			}
		}
	}
}
