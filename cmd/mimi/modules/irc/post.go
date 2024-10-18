package irc

import (
	"context"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"
	"within.website/x/proto/mimi/announce"
)

type PostService struct {
	*Module
}

func (a *PostService) Post(ctx context.Context, su *announce.StatusUpdate) (*emptypb.Empty, error) {
	a.conn.Privmsgf(*ircChannel, "%s", su.Body)

	if _, err := a.dg.ChannelMessageSend(*discordPOSSEAnnounceChannel, su.Body); err != nil {
		return nil, twirp.InternalErrorWith(err).WithMeta("step", "discord")
	}

	return &emptypb.Empty{}, nil
}
