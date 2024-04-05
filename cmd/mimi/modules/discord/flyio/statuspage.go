package flyio

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/protobuf/types/known/emptypb"
	"within.website/x/proto/mimi/statuspage"
)

var (
	statusChannelID = flag.String("fly-discord-status-channel", "1194719777625751573", "fly discord status channel ID")
)

func (m *Module) RegisterHTTP(mux *http.ServeMux) {
	mux.Handle(statuspage.UpdatePathPrefix, statuspage.NewUpdateServer(&UpdateService{Module: m}))
}

type UpdateService struct {
	*Module
}

func (u *UpdateService) Poke(ctx context.Context, sp *statuspage.StatusUpdate) (*emptypb.Empty, error) {
	go u.UpdateStatus(context.Background(), sp)

	return &emptypb.Empty{}, nil
}

func (u *UpdateService) UpdateStatus(ctx context.Context, sp *statuspage.StatusUpdate) {
	if sp.GetIncident() != nil {
		u.UpdateIncident(ctx, sp.GetIncident())
		return
	}

	u.UpdateComponent(ctx, sp)
}

func (u *UpdateService) UpdateIncident(ctx context.Context, incident *statuspage.Incident) {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s\nimpact: %s\n", incident.GetName(), incident.GetImpact())
	fmt.Fprintf(&sb, "Follow the incident at %s\n", incident.GetShortlink())
	fmt.Fprintf(&sb, "Status: %s\n", incident.GetStatus())
	fmt.Fprintf(&sb, "Latest update:\n")
	fmt.Fprintf(&sb, "At %s: \n%s\n\n", incident.GetIncidentUpdates()[0].GetCreatedAt(), incident.GetIncidentUpdates()[0].GetBody())

	if _, err := u.sess.ChannelMessageSendEmbed(*statusChannelID, &discordgo.MessageEmbed{
		URL:         incident.GetShortlink(),
		Title:       fmt.Sprintf("Incident %s: %s", incident.GetId(), incident.GetName()),
		Description: sb.String(),
	}, discordgo.WithContext(ctx)); err != nil {
		slog.Error("failed to send incident update", "err", err, "incident", incident)
		return
	}
}

func (u *UpdateService) UpdateComponent(ctx context.Context, sp *statuspage.StatusUpdate) {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s\nstatus: %s\n", sp.GetComponent().GetName(), sp.GetComponent().GetStatus())

	if _, err := u.sess.ChannelMessageSendEmbed(*statusChannelID, &discordgo.MessageEmbed{
		URL:         "https://status.flyio.net/",
		Title:       fmt.Sprintf("Component %s: %s", sp.GetComponent().GetId(), sp.GetComponent().GetName()),
		Description: sb.String(),
	}); err != nil {
		slog.Error("failed to send component update", "err", err, "component", sp.GetComponent())
		return
	}
}
