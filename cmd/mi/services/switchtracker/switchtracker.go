package switchtracker

import (
	"context"
	"errors"
	"log/slog"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
	"within.website/x/cmd/mi/models"
	pb "within.website/x/proto/mi"
)

type SwitchTracker struct {
	db  *gorm.DB
	dao *models.DAO
}

func New(dao *models.DAO) *SwitchTracker {
	return &SwitchTracker{
		dao: dao,
	}
}

func (s *SwitchTracker) Members(ctx context.Context, _ *emptypb.Empty) (*pb.MembersResp, error) {
	members, err := s.dao.Members(ctx)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	var resp pb.MembersResp
	for _, m := range members {
		resp.Members = append(resp.Members, m.AsProto())
	}

	return &resp, nil
}

func (s *SwitchTracker) WhoIsFront(ctx context.Context, _ *emptypb.Empty) (*pb.FrontChange, error) {
	sw, err := s.dao.WhoIsFront(ctx)
	if err != nil {
		slog.Error("can't find who is front", "err", err)
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("can't find current switch")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	slog.Info("current front", "sw", sw)

	return sw.AsFrontChange(), nil
}

func (s *SwitchTracker) Switch(ctx context.Context, req *pb.SwitchReq) (*pb.SwitchResp, error) {
	if err := req.Valid(); err != nil {
		slog.Error("can't switch without a member", "req", req, "err", err)
		return nil, twirp.InvalidArgumentError("member_name", err.Error())
	}

	old, new, err := s.dao.SwitchFront(ctx, req.GetMemberName())
	if err != nil {
		slog.Error("can't switch front", "req", req, "err", err)
		switch {
		case errors.Is(err, models.ErrCantSwitchToYourself):
			twirp.InvalidArgumentError("member_name", "cannot switch to yourself").
				WithMeta("member_name", req.GetMemberName())
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("can't find current switch")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	return &pb.SwitchResp{
		Old:     old.AsProto(),
		Current: new.AsProto(),
	}, nil
}

func (s *SwitchTracker) GetSwitch(ctx context.Context, req *pb.GetSwitchReq) (*pb.FrontChange, error) {
	if err := req.Valid(); err != nil {
		slog.Error("can't get switch by ID", "req", req, "err", err)
		return nil, twirp.InvalidArgumentError("id", err.Error())
	}

	sw, err := s.dao.GetSwitch(ctx, req.GetId())
	if err != nil {
		slog.Error("can't get switch", "req", req, "err", err)
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("can't find switch").
				WithMeta("id", req.GetId())
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	return sw.AsFrontChange(), nil
}

func (s *SwitchTracker) ListSwitches(ctx context.Context, req *pb.ListSwitchesReq) (*pb.ListSwitchesResp, error) {
	var switches []models.Switch

	if req.GetCount() == 0 {
		req.Count = 30
	}

	switches, err := s.dao.ListSwitches(ctx, int(req.GetCount()), int(req.GetPage()))
	if err != nil {
		slog.Error("can't get switches", "req", req, "err", err)
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("can't find switch info")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	if len(switches) == 0 {
		return nil, twirp.NotFoundError("no switches returned")
	}

	var resp pb.ListSwitchesResp

	for _, sw := range switches {
		resp.Switches = append(resp.Switches, sw.AsFrontChange())
	}

	return &resp, nil
}
