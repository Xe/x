package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"expvar"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	irc "github.com/thoj/go-ircevent"
	"go.jetpack.io/tyson"
	"honnef.co/go/transmission"
	"tailscale.com/hostinfo"
	"tailscale.com/jsondb"
	"tailscale.com/tsnet"
	"within.website/x/internal"
	"within.website/x/web/parsetorrentname"
)

var (
	dbLoc        = flag.String("db-loc", "./data.json", "path to data file")
	tysonConfig  = flag.String("tyson-config", "./config.ts", "path to configuration secrets (TySON)")
	externalSeed = flag.Bool("external-seed", false, "try to external seed?")
	tsnetVerbose = flag.Bool("tsnet-verbose", false, "enable verbose tsnet logging")

	crcCheckCLI = flag.Bool("crc-check", false, "if true, check args[0] against hash args[1]")

	annRegex = regexp.MustCompile(`^New Torrent Announcement: <([^>]*)>\s+Name:'(.*)' uploaded by '.*' ?(freeleech)?\s+-\s+https://\w+.\w+.\w+./\w+./([0-9]+)$`)

	snatches = expvar.NewInt("gauge_sanguisuga_snatches")
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
		return nil, errors.New("invalid torrent announcement format")
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

	if *crcCheckCLI {
		if flag.NArg() != 2 {
			log.Fatalf("usage: %s <filename> <hash>", os.Args[0])
		}

		fname := flag.Arg(0)
		hash := flag.Arg(1)

		ok, err := crcCheck(fname, hash)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("hash status: %v", ok)
		return
	}

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
			TVSnatches:    map[string]TorrentAnnouncement{},
			AnimeSnatches: map[string]SubspleaseAnnouncement{},
			TVWatch:       c.Shows,
		}
	}

	if db.Data.TVSnatches == nil {
		db.Data.TVSnatches = map[string]TorrentAnnouncement{}
	}

	if db.Data.AnimeSnatches == nil {
		db.Data.AnimeSnatches = map[string]SubspleaseAnnouncement{}
	}

	if len(db.Data.TVWatch) == 0 {
		db.Data.TVWatch = c.Shows
	}

	if err := db.Save(); err != nil {
		log.Fatalf("can't ping database: %v", err)
	}

	var dataDir string
	if c.Tailscale.DataDir != nil {
		dataDir = *c.Tailscale.DataDir
	}

	srv := &tsnet.Server{
		Dir:      dataDir,
		AuthKey:  c.Tailscale.Authkey,
		Hostname: c.Tailscale.Hostname,
		Logf:     func(string, ...any) {},
	}

	if *tsnetVerbose {
		srv.Logf = slog.NewLogLogger(slog.Default().Handler().WithAttrs([]slog.Attr{slog.String("from", "tsnet")}), slog.LevelDebug).Printf
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("can't start tsnet server: %v", err)
	}

	go func() {
		ln, err := srv.Listen("tcp", ":80")
		if err != nil {
			log.Fatalf("can't listen on tsnet: %v", err)
		}
		defer ln.Close()

		log.Fatal(http.Serve(ln, http.DefaultServeMux))
	}()

	cl := &transmission.Client{
		Client:   srv.HTTPClient(),
		Endpoint: c.Transmission.URL,
		Username: c.Transmission.User,
		Password: c.Transmission.Password,
	}

	if _, err := cl.SessionStats(); err != nil {
		log.Fatalf("can't connect to transmission: %v", err)
	}

	bot, err := telego.NewBot(c.Telegram.Token)
	if err != nil {
		log.Fatalf("can't connect to telegram: %v", err)
	}

	defer bot.StopLongPolling()

	s := &Sanguisuga{
		Config: c,
		cl:     cl,
		db:     db,
		bot:    bot,
		tmpl:   template.Must(template.ParseFS(templates, "tmpl/*.html")),

		animeInFlight: map[string]*SubspleaseAnnouncement{},
	}

	http.HandleFunc("/", s.AdminIndex)
	http.HandleFunc("/anime", s.AdminAnimeList)
	http.HandleFunc("/tv", s.AdminTVList)

	http.HandleFunc("/api/anime/list", s.ListAnime)
	http.HandleFunc("/api/anime/snatches", s.ListAnimeSnatches)
	http.HandleFunc("/api/anime/track", s.TrackAnime)
	http.HandleFunc("/api/anime/untrack", s.UntrackAnime)

	http.HandleFunc("/api/tv/list", s.ListTV)
	http.HandleFunc("/api/tv/snatches", s.ListTVSnatches)
	http.HandleFunc("/api/tv/track", s.TrackTV)
	http.HandleFunc("/api/tv/untrack", s.UntrackTV)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))

	go s.XDCC()

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
	dbLock sync.Mutex
	bot    *telego.Bot
	tmpl   *template.Template

	animeInFlight map[string]*SubspleaseAnnouncement
	aifLock       sync.Mutex
}

func (s *Sanguisuga) Notify(msg string) {
	s.bot.SendMessage(tu.Message(tu.ID(s.Config.Telegram.MentionUser), msg))
}

type State struct {
	TVSnatches map[string]TorrentAnnouncement
	TVWatch    []Show

	AnimeSnatches map[string]SubspleaseAnnouncement
	AnimeWatch    []Show
}

func (s *Sanguisuga) ExternalSeedAnime(ta *TorrentAnnouncement, lg *slog.Logger) {
	fname := fmt.Sprintf("%s.mkv", ta.Category)
	_ = fname
	// make goroutine to delay until download is done, set up directory structure, hard link, and download torrent

	for {
		s.aifLock.Lock()
		_, ok := s.animeInFlight[fname]
		s.aifLock.Unlock()
		if ok {
			time.Sleep(5 * time.Second)
			continue
		} else {
			break
		}
	}

	s.dbLock.Lock()
	ann, ok := s.db.Data.AnimeSnatches[fname]
	s.dbLock.Unlock()

	if !ok {
		lg.Debug("can't opportunistically external seed", "why", "episode not already snatched")
		return
	}

	s.dbLock.Lock()
	var show Show
	for _, trackShow := range s.db.Data.AnimeWatch {
		if trackShow.Title == ann.ShowName {
			show = trackShow
		}
	}
	s.dbLock.Unlock()

	if show.Title == "" {
		lg.Debug("can't opportunistically external seed", "why", "can't find show in database but we have a snatch?")
		return
	}

	torrentURL := ConvertURL(ta.TorrentID, s.Config.RSSKey, ta.Name)

	dirName := filepath.Join("/data", "Torrents", "seedHacking", "tl", ta.TorrentID)

	if err := os.Link(filepath.Join(show.DiskPath, fname), filepath.Join(dirName, fname)); err != nil {
		lg.Error("can't set up seedhacking directory", "err", err)
		return
	}

	var buf bytes.Buffer
	resp, err := http.Get(torrentURL)
	if err != nil {
		lg.Error("can't download torrent", "url", torrentURL, "err", err, "torrentID", ta.TorrentID)
		return
	}

	if resp.StatusCode != http.StatusOK {
		lg.Error("got wrong status code", "want", http.StatusOK, "got", resp.StatusCode, "url", torrentURL, "torrentID", ta.TorrentID)
		return
	}

	defer resp.Body.Close()
	if n, err := io.Copy(&buf, resp.Body); err != nil {
		lg.Error("can't fetch torrent body", "url", torrentURL, "err", err, "torrentID", ta.TorrentID)
		return
	} else {
		lg.Info("downloaded bytes", "n", n, "url", torrentURL)
	}

	metaInfo := base64.StdEncoding.EncodeToString(buf.Bytes())

	_, _, err = s.cl.AddTorrent(&transmission.NewTorrent{
		DownloadDir: dirName,
		Metainfo:    metaInfo,
		Paused:      false,
	})
	if err != nil {
		lg.Error("error adding torrent", "url", torrentURL, "err", err, "torrentID", ta.TorrentID)
		return
	}

	lg.Info("opportunistically external seeding")
	snatches.Add(1)
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

	lg := slog.Default().With("category", ta.Category, "freeleech", ta.Freeleech, "name", ta.Name)

	lg.Debug("found torrent announcment")

	switch ta.Category {
	case "Animation :: Anime":
		if !*externalSeed {
			return
		}

		go s.ExternalSeedAnime(ta, lg)
	case "TV :: Episodes HD":
		ti, err := parsetorrentname.Parse(ta.Name)
		if err != nil {
			lg.Error("can't parse ShowMeta", "err", err)
			return
		}
		id := fmt.Sprintf("S%02dE%02d", ti.Season, ti.Episode)
		lg := lg.With("title", ti.Title, "id", id, "quality", ti.Resolution, "group", ti.Group)
		lg.Debug("found ShowMeta")

		stateKey := fmt.Sprintf("%s %s", ti.Title, id)

		s.dbLock.Lock()
		defer s.dbLock.Unlock()

		for _, show := range s.db.Data.TVWatch {
			if s.db.Data == nil {
				s.db.Data = &State{
					TVSnatches: map[string]TorrentAnnouncement{},
				}
			}
			if _, found := s.db.Data.TVSnatches[stateKey]; found {
				lg.Info("already snatched", "title", ti.Title, "id", id)
				return
			}

			if show.Title != ti.Title {
				lg.Debug("wrong name")
				continue
			}

			if show.Quality != ti.Resolution {
				lg.Debug("wrong resolution")
				continue
			}

			torrentURL := ConvertURL(ta.TorrentID, s.Config.RSSKey, ta.Name)

			lg.Debug("found url", "url", torrentURL)
			downloadDir := filepath.Join(show.DiskPath, fmt.Sprintf("Season %02d", ti.Season))

			var buf bytes.Buffer
			resp, err := http.Get(torrentURL)
			if err != nil {
				lg.Error("can't download torrent", "url", torrentURL, "err", err, "torrentID", ta.TorrentID)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				lg.Error("got wrong status code", "want", http.StatusOK, "got", resp.StatusCode, "url", torrentURL, "torrentID", ta.TorrentID)
				continue
			}

			defer resp.Body.Close()
			if n, err := io.Copy(&buf, resp.Body); err != nil {
				lg.Error("can't fetch torrent body", "url", torrentURL, "err", err, "torrentID", ta.TorrentID)
				continue
			} else {
				lg.Info("downloaded bytes", "n", n, "url", torrentURL)
			}

			metaInfo := base64.StdEncoding.EncodeToString(buf.Bytes())

			t, dupe, err := s.cl.AddTorrent(&transmission.NewTorrent{
				DownloadDir: downloadDir,
				Metainfo:    metaInfo,
				Paused:      false,
			})
			if err != nil {
				lg.Error("error adding torrent", "url", torrentURL, "err", err, "torrentID", ta.TorrentID)
				return
			}

			lg.Info("added torrent", "title", ti.Title, "id", id, "path", downloadDir, "infohash", t.Hash, "tid", t.ID, "dupe", dupe)
			snatches.Add(1)

			s.Notify(fmt.Sprintf("added torrent for %s %s", ti.Title, id))

			s.db.Data.TVSnatches[stateKey] = *ta
			if err := s.db.Save(); err != nil {
				lg.Error("error saving state", "err", err)
			}
		}
	}
}
