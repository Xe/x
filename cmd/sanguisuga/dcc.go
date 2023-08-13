package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	irc "github.com/thoj/go-ircevent"
	"golang.org/x/exp/slog"
	"tailscale.com/metrics"
	"within.website/x/cmd/sanguisuga/dcc"
)

var (
	subspleaseAnnounceRegex = regexp.MustCompile(`^.*\* (\[SubsPlease\] (.*) - ([0-9]+) \(([0-9]+p)\).*\.mkv) \* /MSG ([A-Za-z-_|]+) XDCC SEND ([0-9]+)$`)
	dccCommand              = regexp.MustCompile(`^DCC SEND "(.*)" ([0-9]+) ([0-9]+) ([0-9]+)$`)

	bytesDownloaded = &metrics.LabelMap{Label: "filename"}
)

func init() {
	expvar.Publish("gauge_sanguisuga_bytes_downloaded", bytesDownloaded)
}

type SubspleaseAnnouncement struct {
	Filename, ShowName string
	Episode, Quality   string
	BotName, PackID    string
}

func (sa SubspleaseAnnouncement) Key() string {
	return fmt.Sprintf("%s %s %s", sa.BotName, sa.ShowName, sa.Episode)
}

func (sa SubspleaseAnnouncement) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("showname", sa.ShowName),
		slog.String("episode", sa.Episode),
		slog.String("quality", sa.Quality),
		slog.String("botName", sa.BotName),
	)
}

func ParseSubspleaseAnnouncement(input string) (*SubspleaseAnnouncement, error) {
	match := subspleaseAnnounceRegex.FindStringSubmatch(input)

	if match == nil {
		return nil, errors.New("invalid annoucement format")
	}

	return &SubspleaseAnnouncement{
		Filename: match[1],
		ShowName: match[2],
		Episode:  match[3],
		Quality:  match[4],
		BotName:  match[5],
		PackID:   match[6],
	}, nil
}

func int2ip(nn uint32) string {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip.String()
}

func (s *Sanguisuga) XDCC() {
	ircCli := irc.IRC(s.Config.XDCC.Nick, s.Config.XDCC.User)
	ircCli.Password = s.Config.XDCC.Password
	ircCli.RealName = s.Config.XDCC.Real
	ircCli.AddCallback("001", func(ev *irc.Event) {
		ircCli.Join(s.Config.XDCC.Channel)
	})
	ircCli.AddCallback("PRIVMSG", s.ScrapeSubsplease)
	ircCli.AddCallback("CTCP", s.SubspleaseDCC)

	ircCli.Log = slog.NewLogLogger(slog.Default().Handler().WithAttrs([]slog.Attr{slog.String("from", "ircevent"), slog.String("for", "anime")}), slog.LevelInfo)
	ircCli.Timeout = 5 * time.Second

	if err := ircCli.Connect(s.Config.XDCC.Server); err != nil {
		log.Fatalf("can't connect to XDCC server %s: %v", s.Config.XDCC.Server, err)
	}

	ircCli.Loop()
}

func (s *Sanguisuga) TrackAnime(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var show Show
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&show)
	if err != nil {
		slog.Error("can't read request body", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	s.db.Data.AnimeWatch = append(s.db.Data.AnimeWatch, show)
	if err := s.db.Save(); err != nil {
		slog.Error("can't save database", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(s.db.Data.AnimeWatch)
}

func (s *Sanguisuga) ListAnimeSnatches(w http.ResponseWriter, r *http.Request) {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	json.NewEncoder(w).Encode(s.db.Data.AnimeSnatches)
}

func (s *Sanguisuga) ListAnime(w http.ResponseWriter, r *http.Request) {
	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	json.NewEncoder(w).Encode(s.db.Data.AnimeWatch)
}

func (s *Sanguisuga) ScrapeSubsplease(ev *irc.Event) {
	if ev.Code != "PRIVMSG" {
		return
	}

	if ev.Arguments[0] != s.Config.XDCC.Channel {
		return
	}

	switch ev.Nick {
	case "CR-ARUTHA|NEW", "CR-HOLLAND|NEW", "Belath":
	default:
		return
	}

	ann, err := ParseSubspleaseAnnouncement(ev.MessageWithoutFormat())
	if err != nil {
		slog.Debug("can't parse announcement", "input", ev.MessageWithoutFormat(), "err", err)
		return
	}

	if ann.Quality != "1080p" {
		return
	}

	lg := slog.Default().With("announcement", ann)
	lg.Info("found announcement")

	s.dbLock.Lock()
	defer s.dbLock.Unlock()

	s.aifLock.Lock()
	defer s.aifLock.Unlock()

	if _, ok := s.db.Data.AnimeSnatches[ann.Filename]; ok {
		return
	}

	found := false
	for _, show := range s.db.Data.AnimeWatch {
		if ann.ShowName == show.Title {
			found = true
		}
	}

	if !found {
		return
	}

	// if already being fetched, don't fetch again
	if _, ok := s.animeInFlight[ann.Filename]; ok {
		return
	}

	ev.Connection.Privmsgf(ann.BotName, "XDCC SEND %s", ann.PackID)

	s.animeInFlight[ann.Filename] = ann

	s.db.Data.AnimeSnatches[ann.Filename] = *ann
	if err := s.db.Save(); err != nil {
		lg.Error("can't save database", "err", err)
		return
	}
}

func (s *Sanguisuga) SubspleaseDCC(ev *irc.Event) {
	matches := dccCommand.FindStringSubmatch(ev.MessageWithoutFormat())
	if matches == nil {
		return
	}

	s.aifLock.Lock()
	defer s.aifLock.Unlock()

	if len(matches) != 5 {
		slog.Error("wrong message from DCC bot", "botName", ev.Nick, "message", ev.Message())
		return
	}

	fname := matches[1]
	ipString := matches[2]
	port := matches[3]
	sizeString := matches[4]

	if strings.HasSuffix(fname, "\"") {
		fname = fname[:len(fname)-2]
	}

	ann, _ := s.animeInFlight[fname]

	if ann == nil {
		slog.Debug("ann == nil?", "fname", fname, "ann", s.animeInFlight[fname])
		return
	}

	ipUint, err := strconv.ParseUint(ipString, 10, 32)
	if err != nil {
		slog.Error("can't parse IP address", "addr", ipString, "err", err)
		return
	}

	ip := int2ip(uint32(ipUint))
	addr := net.JoinHostPort(ip, port)

	size, err := strconv.Atoi(sizeString)
	if err != nil {
		slog.Error("can't parse size", "size", sizeString, "err", err)
		return
	}

	lg := slog.Default().With("fname", fname, "botName", ev.Nick, "addr", addr)
	lg.Info("fetching episode")

	baseDir := ""
	for _, show := range s.db.Data.AnimeWatch {
		if ann.ShowName == show.Title {
			baseDir = show.DiskPath
		}
	}

	outFname := filepath.Join(baseDir, fname)

	os.MkdirAll(baseDir, 0777)

	fout, err := os.Create(outFname)
	if err != nil {
		lg.Error("can't create output file", "outFname", outFname, "err", err)
		return
	}
	defer fout.Close()

	d := dcc.NewDCC(addr, size, fout)

	ctx, cancel := context.WithTimeout(ev.Ctx, 120*time.Minute)
	defer cancel()

	start := time.Now()
	progc, errc := d.Run(ctx)

outer:
	for {
		select {
		case p := <-progc:
			curr := bytesDownloaded.GetFloat(fname)
			curr.Set(p.CurrentFileSize)

			if p.CurrentFileSize == p.FileSize {
				break outer
			}

			if p.Percentage >= 100 {
				break outer
			}

			lg.Debug("download progress", "progress", p)
		case err := <-errc:
			lg.Error("error in DCC thread, giving up", "err", err)
			delete(s.animeInFlight, fname)
			return
		}
	}

	delete(s.animeInFlight, fname)
	dur := time.Since(start)
	lg.Info("finished downloading", "dur", dur.String())
}
