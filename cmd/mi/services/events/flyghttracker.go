package events

import (
	"context"
	"log/slog"

	"within.website/x/cmd/mi/services/events/flyghttracker"
	pb "within.website/x/proto/mi"
)

func (e *Events) syndicateToFlyghtTracker(ctx context.Context, ev *pb.Event) error {
	if !ev.GetSyndicate() {
		slog.Debug("not syndicating event to flyght tracker", "event", ev)
		return nil
	}

	ftev := flyghttracker.Event{
		Name: ev.GetName(),
		URL:  ev.GetUrl(),
		StartDate: flyghttracker.Date{
			Time: ev.GetStartDate().AsTime(),
		},
		Location: ev.GetLocation(),
		People:   []string{"Xe"},
	}

	if ev.GetStartDate().Seconds != ev.GetEndDate().Seconds {
		ftev.EndDate = flyghttracker.Date{
			Time: ev.GetEndDate().AsTime(),
		}
	}

	if err := e.flyghtTracker.Create(ctx, ftev); err != nil {
		return err
	}

	return nil
}
