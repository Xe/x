package irc

import (
	"context"
	"flag"
	"net/http"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"
	jsonfeedv1 "within.website/x/gen/within/website/x/external/jsonfeed/v1"
	announcev1 "within.website/x/gen/within/website/x/mimi/announce/v1"
	xinternal "within.website/x/internal"
)

var (
	discordPOSSEAnnounceChannel = flag.String("discord-posse-announce-channel", "1191450220731584532", "Discord channel to announce in")

	ircAnnouncerUsername = flag.String("irc-announcer-username", "mimi", "IRC announcer HTTP username")
	ircAnnouncerPassword = flag.String("irc-announcer-password", "", "IRC announcer HTTP password")
)

func (m *Module) RegisterHTTP(mux *http.ServeMux) {
	var h http.Handler = announcev1.NewAnnounceServer(&AnnounceService{Module: m})
	h = xinternal.PasswordMiddleware(*ircAnnouncerUsername, *ircAnnouncerPassword, h)
	var postH http.Handler = announcev1.NewPostServer(&PostService{Module: m})
	postH = xinternal.PasswordMiddleware(*ircAnnouncerUsername, *ircAnnouncerPassword, postH)

	mux.Handle(announcev1.AnnouncePathPrefix, h)
	mux.Handle(announcev1.PostPathPrefix, postH)
}

type AnnounceService struct {
	*Module
	announcev1.UnimplementedAnnounceServer
}

type State struct {
	Announced map[string]struct{}
}

func (a *AnnounceService) Announce(ctx context.Context, msg *jsonfeedv1.Item) (*emptypb.Empty, error) {
	a.conn.Privmsgf(*ircChannel, "New article :: %s :: %s", msg.Title, msg.Url)

	if _, err := a.dg.ChannelMessageSend(*discordPOSSEAnnounceChannel, msg.GetUrl()); err != nil {
		return nil, twirp.InternalErrorWith(err).WithMeta("step", "discord")
	}

	return &emptypb.Empty{}, nil
}
