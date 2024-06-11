package models

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	pb "within.website/x/proto/mi"
)

// Event represents an event that members of DevRel will be attending.
type Event struct {
	gorm.Model
	ID          int `gorm:"primaryKey"`
	Name        string
	URL         string
	StartDate   time.Time
	EndDate     time.Time
	Location    string `gorm:"index"`
	Description string
	Syndicate   bool
}

func (e *Event) AsProto() *pb.Event {
	return &pb.Event{
		Id:          int32(e.ID),
		Name:        e.Name,
		Url:         e.URL,
		StartDate:   timestamppb.New(e.StartDate),
		EndDate:     timestamppb.New(e.EndDate),
		Location:    e.Location,
		Description: e.Description,
		Syndicate:   e.Syndicate,
	}
}

func (d *DAO) CreateEvent(ctx context.Context, event *Event) (*Event, error) {
	return event, d.db.WithContext(ctx).Create(event).Error
}

func (d *DAO) GetEvent(ctx context.Context, id int) (*Event, error) {
	var event Event
	if err := d.db.WithContext(ctx).Where("id = ?", id).First(&event).Error; err != nil {
		return nil, err
	}

	return &event, nil
}

func (d *DAO) UpcomingEvents(ctx context.Context, count int) ([]Event, error) {
	var events []Event
	if err := d.db.
		WithContext(ctx).
		Where("end_date >= ?", time.Now()).
		Limit(count).
		Order("start_date").
		Find(&events).Error; err != nil {
		return nil, err
	}

	return events, nil
}
