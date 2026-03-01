package models

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	pb "within.website/x/gen/within/website/x/mi/v1"
)

// Member is a member of the Within system.
type Member struct {
	ID        int        // unique number to use as a primary key
	Name      string     `gorm:"uniqueIndex"` // the name of the member
	AvatarURL string     // public URL to the member's avatar
	Birthday  *time.Time // optional birthday as RFC 3339 timestamp
	Aliases   string     // comma-separated list of alternative names
}

// MatchesName reports whether the given name matches this member's
// canonical name or any of their aliases (case-insensitive).
func (m Member) MatchesName(name string) bool {
	if strings.EqualFold(m.Name, name) {
		return true
	}
	for _, alias := range strings.Split(m.Aliases, ",") {
		if alias = strings.TrimSpace(alias); alias != "" && strings.EqualFold(alias, name) {
			return true
		}
	}
	return false
}

// AsProto converts a Member to its protobuf representation.
func (m Member) AsProto() *pb.Member {
	protoMember := &pb.Member{
		Id:        int32(m.ID),
		Name:      m.Name,
		AvatarUrl: m.AvatarURL,
	}

	if m.Birthday != nil {
		protoMember.Birthday = timestamppb.New(*m.Birthday)
	}

	return protoMember
}
