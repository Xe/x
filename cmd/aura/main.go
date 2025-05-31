package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/bwmarrin/discordgo"
	hashids "github.com/speps/go-hashids"
	"within.website/x/internal"
	"within.website/x/internal/pvfm/bot"
	"within.website/x/internal/pvfm/commands/source"
	"within.website/x/internal/pvfm/recording"
	"within.website/x/xess"
)

//go:generate go tool templ generate

var (
	token           = flag.String("token", "", "Token for authentication")
	dataPrefix      = flag.String("data-prefix", "", "Data prefix")
	recordingDomain = flag.String("recording-domain", "", "Recording domain")
	hashidsSalt     = flag.String("hashids-salt", "", "Salt for Hashids")
	port            = flag.String("port", "8080", "Port number")

	//go:embed aura.webp
	static embed.FS
)

func main() {
	internal.HandleStartup()

	dg, err := discordgo.New("Bot " + *token)
	if err != nil {
		log.Fatal(err)
	}

	hid := hashids.NewData()
	hid.Salt = *hashidsSalt

	hiid, err := hashids.NewWithData(hid)
	if err != nil {
		log.Fatal(err)
	}

	a := &aura{
		cs:              bot.NewCommandSet(),
		s:               dg,
		guildRecordings: map[string]*rec{},

		hid: hiid,

		state: &state{
			DownloadURLs: map[string]string{},
			PermRoles:    map[string]string{},
			Shorturls:    map[string]string{},
		},
	}

	err = a.state.Load()
	if err != nil {
		log.Println(err)
	}

	a.cs.AddCmd("roles", "", bot.NoPermissions, a.roles)
	a.cs.AddCmd("setup", setupHelp, bot.NoPermissions, a.setup)
	a.cs.AddCmd("djon", djonHelp, a.Permissons, a.djon)
	a.cs.AddCmd("djoff", djoffHelp, a.Permissons, a.djoff)
	a.cs.AddCmd("source", "Source code information", bot.NoPermissions, source.Source)

	dg.AddHandler(a.Handle)

	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("ready")

	mux := http.NewServeMux()

	mux.Handle("/id/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		redir, ok := a.state.Shorturls[id]
		if !ok {
			http.Error(w, "not found, sorry", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, redir, http.StatusFound)
	}))

	xess.Mount(mux)

	mux.HandleFunc("/links.json", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(a.state.Shorturls)
	})

	mux.Handle("/BronyRadio/", http.FileServer(http.Dir("/data")))
	mux.Handle("/sleepypony/", http.FileServer(http.Dir("/data")))
	mux.Handle("/toastbeard/", http.FileServer(http.Dir("/data")))
	mux.Handle("/var/", http.FileServer(http.Dir("/data")))
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	mux.Handle("/{$}", templ.Handler(
		xess.Base(
			"PonyvilleFM DJ recording archives",
			nil,
			nil,
			index(),
			nil,
		),
	))
	mux.Handle("/", templ.Handler(
		xess.Simple("Not found", notFound()),
		templ.WithStatus(http.StatusNotFound),
	))

	slog.Info("listening on", "url", fmt.Sprintf("http://localhost:%s", *port))
	log.Fatal(http.ListenAndServe(":"+*port, mux))
}

func genFname(username string) (string, error) {
	return fmt.Sprintf("%s - %s.mp3", username, time.Now().Format(time.RFC3339)), nil
}

type aura struct {
	cs *bot.CommandSet
	s  *discordgo.Session

	guildRecordings map[string]*rec
	state           *state
	hid             *hashids.HashID
}

type state struct {
	DownloadURLs map[string]string // Guild ID -> URL
	PermRoles    map[string]string // Guild ID -> needed role ID
	Shorturls    map[string]string // hashid -> partial route
}

func (s *state) Save() error {
	fout, err := os.Create(path.Join(*dataPrefix, "state.json"))
	if err != nil {
		return err
	}
	defer fout.Close()

	return json.NewEncoder(fout).Encode(s)
}

func (s *state) Load() error {
	fin, err := os.Open(path.Join(*dataPrefix, "state.json"))
	if err != nil {
		return err
	}
	defer fin.Close()

	return json.NewDecoder(fin).Decode(s)
}

type rec struct {
	*recording.Recording
	creator string
}

const (
	djonHelp  = `Start a DJ set recording`
	djoffHelp = `Stop a DJ set recording`
	setupHelp = `Set up the bot for your guild`
)

func (a *aura) Permissons(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		return err
	}

	gid := ch.GuildID
	role := a.state.PermRoles[gid]

	gu, err := s.GuildMember(gid, m.Author.ID)
	if err != nil {
		return err
	}

	slog.Info("want role", "role", role, "author_roles", gu.Roles)

	found := false
	for _, r := range gu.Roles {
		if r == role {
			found = true
			break
		}
	}

	if !found {
		return errors.New("aura: no permissions")
	}

	return nil
}

func (a *aura) roles(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		return err
	}

	gid := ch.GuildID

	result := "Roles in this group:\n"

	roles, err := s.GuildRoles(gid)
	if err != nil {
		return err
	}

	for _, r := range roles {
		result += fmt.Sprintf("- %s: %s\n", r.ID, r.Name)
	}

	s.ChannelMessageSend(m.ChannelID, result)
	return nil
}

func (a *aura) setup(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	if len(parv) != 3 {
		return errors.New("aura: wrong number of params for setup")
	}

	role := parv[1]
	url := parv[2]

	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		return err
	}

	gid := ch.GuildID

	roles, err := s.GuildRoles(gid)
	if err != nil {
		return err
	}

	found := false
	for _, r := range roles {
		if r.ID == role {
			found = true
			break
		}
	}

	if !found {
		return errors.New("aura: Role not found")
	}

	a.state.PermRoles[gid] = role
	a.state.DownloadURLs[gid] = url

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Guild %s set up for recording url %s controlled by role %s", gid, url, role))

	a.state.Save()
	return nil
}

func (a *aura) djon(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		return err
	}

	gid := ch.GuildID
	creator := m.Author.Username

	member, err := s.GuildMember(gid, m.Author.ID)
	if err != nil {
		return err
	}

	if member.Nick != "" {
		creator = member.Nick
	}

	fname, err := genFname(creator)
	if err != nil {
		return err
	}

	_, ok := a.guildRecordings[gid]
	if ok {
		log.Println(a.guildRecordings)
		return errors.New("aura: another recording is already in progress")
	}

	os.Mkdir(path.Join(*dataPrefix, gid), 0775)

	rr, err := recording.New(a.state.DownloadURLs[gid], filepath.Join(*dataPrefix, gid, fname))
	if err != nil {
		return err
	}

	a.guildRecordings[gid] = &rec{
		Recording: rr,
		creator:   creator,
	}

	go func() {
		err := rr.Start()
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("recording error: %v", err))
			return
		}
	}()

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Now recording: `%s`\n\n%s get in here fam", fname, os.Getenv("NOTIFICATION_SQUAD_ID")))

	go a.waitAndAnnounce(s, m, a.guildRecordings[gid], gid)

	return nil
}

func (a *aura) waitAndAnnounce(s *discordgo.Session, m *discordgo.Message, r *rec, gid string) {
	<-r.Done()

	defer delete(a.guildRecordings, gid)

	fname := r.OutputFilename()
	parts := strings.Split(fname, "/")

	slog.Info("stuff", "fname", fname, "parts", parts)

	recurl := fmt.Sprintf("https://%s/var/%s/%s", *recordingDomain, parts[3], urlencode(parts[4]))
	id, err := a.hid.EncodeInt64([]int64{int64(rand.Int())})
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("This state should be impossible. Recording saved but unknown short URL: %v", err))
		return
	}

	a.state.Shorturls[id] = recurl
	a.state.Save()

	slink := fmt.Sprintf("https://%s/id/%s", *recordingDomain, id)

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Recording complete (%s): %s", time.Since(r.StartTime()).String(), slink))
}

func urlencode(inp string) string {
	return (&url.URL{Path: inp}).String()
}

func (a *aura) djoff(s *discordgo.Session, m *discordgo.Message, parv []string) error {
	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		return err
	}

	gid := ch.GuildID

	r, ok := a.guildRecordings[gid]
	if r == nil || !ok {
		log.Println(a.guildRecordings)
		return errors.New("aura: no recording is currently in progress")
	}

	if r.Err == nil {
		s.ChannelMessageSend(m.ChannelID, "Finishing recording (waiting 30 seconds)")
		time.Sleep(30 * time.Second)

		r.Cancel()
	}

	return nil
}

func (a *aura) Handle(s *discordgo.Session, m *discordgo.MessageCreate) {
	err := a.cs.Run(s, m.Message)
	if err != nil {
		log.Println(err)
	}
}
