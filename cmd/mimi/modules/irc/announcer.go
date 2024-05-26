package irc

import (
	"context"
	"flag"
	"net/http"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"
	xinternal "within.website/x/internal"
	"within.website/x/proto/external/jsonfeed"
	"within.website/x/proto/mimi/announce"
)

var (
	discordPOSSEAnnounceChannel = flag.String("discord-posse-announce-channel", "1191450220731584532", "Discord channel to announce in")

	ircAnnouncerUsername = flag.String("irc-announcer-username", "mimi", "IRC announcer HTTP username")
	ircAnnouncerPassword = flag.String("irc-announcer-password", "", "IRC announcer HTTP password")
)

func (m *Module) RegisterHTTP(mux *http.ServeMux) {
	var h http.Handler = announce.NewAnnounceServer(&AnnounceService{Module: m})
	h = xinternal.PasswordMiddleware(*ircAnnouncerUsername, *ircAnnouncerPassword, h)

	mux.Handle(announce.AnnouncePathPrefix, h)
}

type AnnounceService struct {
	*Module
}

type State struct {
	Announced map[string]struct{}
}

func (a *AnnounceService) Announce(ctx context.Context, msg *jsonfeed.Item) (*emptypb.Empty, error) {
	a.conn.Privmsgf(*ircChannel, "New article :: %s :: %s", msg.Title, msg.Url)

	if _, err := a.dg.ChannelMessageSend(*discordPOSSEAnnounceChannel, msg.GetUrl()); err != nil {
		return nil, twirp.InternalErrorWith(err).WithMeta("step", "discord")
	}

	return &emptypb.Empty{}, nil
}
