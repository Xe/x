package events

import (
	"context"
	"errors"
	"log/slog"

	_ "github.com/lib/pq"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"within.website/x/cmd/mi/models"
	pb "within.website/x/gen/within/website/x/mi/v1"
)

type Events struct {
	dao *models.DAO

	pb.UnimplementedEventsServer
}

var _ pb.Events = &Events{}

// New creates a new Events service.
func New(dao *models.DAO, flyghtTrackerURL string) *Events {
	result := &Events{
		dao: dao,
	}

	return result
}

// Get fetches upcoming events.
func (e *Events) Get(ctx context.Context, _ *emptypb.Empty) (*pb.EventFeed, error) {
	events, err := e.dao.UpcomingEvents(ctx, 10)
	if err != nil {
		slog.Error("can't fetch upcoming events", "err", err)
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("can't find any events")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	if len(events) == 0 {
		return nil, twirp.NotFoundError("can't find any events")
	}

	var pbEvents []*pb.Event
	for _, event := range events {
		pbEvents = append(pbEvents, event.AsProto())
	}

	return &pb.EventFeed{Events: pbEvents}, nil
}

// Add adds a new event to the database.
func (e *Events) Add(ctx context.Context, ev *pb.Event) (*emptypb.Empty, error) {
	event := &models.Event{
		Name:        ev.Name,
		URL:         ev.Url,
		StartDate:   ev.StartDate.AsTime(),
		EndDate:     ev.EndDate.AsTime(),
		Location:    ev.Location,
		Description: ev.Description,
	}

	_, err := e.dao.CreateEvent(ctx, event)
	if err != nil {
		return nil, err
	}

	slog.Info("tracking new event", "event", event)

	return &emptypb.Empty{}, nil
}

func (e *Events) Remove(ctx context.Context, req *pb.Event) (*emptypb.Empty, error) {
	err := e.dao.RemoveEvent(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
