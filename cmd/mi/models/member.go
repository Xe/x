package models

import (
	pb "within.website/x/proto/mi"
)

// Member is a member of the Within system.
type Member struct {
	ID        int    // unique number to use as a primary key
	Name      string `gorm:"uniqueIndex"` // the name of the member
	AvatarURL string // public URL to the member's avatar
}

// AsProto converts a Member to its protobuf representation.
func (m Member) AsProto() *pb.Member {
	return &pb.Member{
		Id:        int32(m.ID),
		Name:      m.Name,
		AvatarUrl: m.AvatarURL,
	}
}
