package models

import (
	"time"

	"gorm.io/gorm"
	pb "within.website/x/proto/mi"
)

// Switch is a record of the system switching front to a different member.
type Switch struct {
	gorm.Model            // adds CreatedAt, UpdatedAt, DeletedAt
	ID         string     `gorm:"uniqueIndex"` // unique identifier for the switch (ULID usually)
	EndedAt    *time.Time // when the switch ends, different from DeletedAt
	MemberID   int        // the member who is now in front
	Member     Member     `gorm:"foreignKey:MemberID"`
}

// AsProto converts a Switch to its protobuf representation.
func (s *Switch) AsProto() *pb.Switch {
	if s == nil {
		return nil
	}

	var endedAt string

	if s.EndedAt != nil {
		endedAt = s.EndedAt.Format(time.RFC3339)
	}

	return &pb.Switch{
		Id:        s.ID,
		StartedAt: s.CreatedAt.Format(time.RFC3339),
		EndedAt:   endedAt,
		MemberId:  int32(s.MemberID),
	}
}

// AsFrontChange converts a Switch to a FrontChange protobuf.
func (s Switch) AsFrontChange() *pb.FrontChange {
	return &pb.FrontChange{
		Member: s.Member.AsProto(),
		Switch: s.AsProto(),
	}
}
