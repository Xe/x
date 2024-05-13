package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"within.website/x/cmd/mi/models"
	pb "within.website/x/proto/mi"
)

func p[T any](v T) *T {
	return &v
}

type SwitchTracker struct {
	db *gorm.DB
}

func NewSwitchTracker(db *gorm.DB) *SwitchTracker {
	return &SwitchTracker{db: db}
}

func (s *SwitchTracker) Members(ctx context.Context, _ *emptypb.Empty) (*pb.MembersResp, error) {
	var members []models.Member
	if err := s.db.Find(&members).Error; err != nil {
		return nil, err
	}

	var resp pb.MembersResp
	for _, m := range members {
		resp.Members = append(resp.Members, m.AsProto())
	}

	return &resp, nil
}

func (s *SwitchTracker) WhoIsFront(ctx context.Context, _ *emptypb.Empty) (*pb.FrontChange, error) {
	var sw models.Switch
	if err := s.db.Joins("Member").Order("created_at DESC").First(&sw).Error; err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return sw.AsFrontChange(), nil
}

func (s *SwitchTracker) Switch(ctx context.Context, req *pb.SwitchReq) (*pb.SwitchResp, error) {
	if err := req.Valid(); err != nil {
		slog.Error("can't switch", "req", req, "err", err)
		return nil, twirp.InvalidArgumentError("member_name", err.Error())
	}

	var sw models.Switch

	tx := s.db.Begin()

	if err := tx.Joins("Member").Where("ended_at IS NULL").First(&sw).Error; err != nil {
		tx.Rollback()
		return nil, twirp.InternalErrorf("failed to find current switch: %w", err)
	}

	if sw.Member.Name == req.MemberName {
		tx.Rollback()
		return nil, twirp.InvalidArgumentError("member_name", "cannot switch to the same member").
			WithMeta("member_name", req.MemberName).
			WithMeta("current_member", sw.Member.Name)
	}

	sw.EndedAt = p(time.Now())
	if err := tx.Save(&sw).Error; err != nil {
		tx.Rollback()
		return nil, twirp.InternalErrorf("failed to save current switch: %w", err)
	}

	var newMember models.Member
	if err := tx.Where("name = ?", req.MemberName).First(&newMember).Error; err != nil {
		tx.Rollback()
		return nil, twirp.NotFoundError("member not found").WithMeta("member_name", req.MemberName)
	}

	newSwitch := models.Switch{
		ID:       ulid.MustNew(ulid.Now(), nil).String(),
		MemberID: newMember.ID,
	}

	if err := tx.Create(&newSwitch).Error; err != nil {
		tx.Rollback()
		return nil, twirp.InternalErrorf("failed to create new switch: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, twirp.InternalErrorf("failed to commit transaction: %w", err)
	}

	slog.Info("switched", "from", sw.AsProto(), "to", newSwitch.AsProto())

	return &pb.SwitchResp{
		Old:     sw.AsProto(),
		Current: newSwitch.AsProto(),
	}, nil
}

func (s *SwitchTracker) GetSwitch(ctx context.Context, req *pb.GetSwitchReq) (*pb.FrontChange, error) {
	if err := req.Valid(); err != nil {
		slog.Error("can't get switch by ID", "req", req, "err", err)
		return nil, twirp.InvalidArgumentError("id", err.Error())
	}

	var sw models.Switch
	if err := s.db.Joins("Member").Where("id = ?", req.Id).First(&sw).Error; err != nil {
		return nil, twirp.NotFoundError("switch not found").WithMeta("id", req.Id)
	}

	return sw.AsFrontChange(), nil
}

func (s *SwitchTracker) ListSwitches(ctx context.Context, req *pb.ListSwitchesReq) (*pb.ListSwitchesResp, error) {
	var switches []models.Switch

	if req.GetCount() == 0 {
		req.Count = 30
	}

	if err := s.db.Joins("Member").Order("rowid DESC").Limit(int(req.GetCount())).Offset(int(req.GetCount() * req.GetPage())).Find(&switches).Error; err != nil {
		return nil, err
	}

	var resp pb.ListSwitchesResp

	for _, sw := range switches {
		resp.Switches = append(resp.Switches, sw.AsFrontChange())
	}

	return &resp, nil
}
