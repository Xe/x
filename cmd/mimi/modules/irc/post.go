package irc

import (
	"context"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"
	announcev1 "within.website/x/gen/within/website/x/mimi/announce/v1"
)

type PostService struct {
	*Module
	announcev1.UnimplementedPostServer
}

func (a *PostService) Post(ctx context.Context, su *announcev1.StatusUpdate) (*emptypb.Empty, error) {
	a.conn.Privmsgf(*ircChannel, "%s", su.Body)

	if _, err := a.dg.ChannelMessageSend(*discordPOSSEAnnounceChannel, su.Body); err != nil {
		return nil, twirp.InternalErrorWith(err).WithMeta("step", "discord")
	}

	return &emptypb.Empty{}, nil
}
