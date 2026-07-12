package users

import (
	"context"
	"errors"
	"log/slog"

	"buf.build/go/protovalidate"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/twitchtv/twirp"
	"gorm.io/gorm"
	"within.website/x/cmd/iamd/models"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

type Server struct {
	dao *models.DAO

	iamv1.UnimplementedUserServiceServer
}

var (
	usersCreated = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "within_website_x",
		Subsystem: "iamd",
		Name:      "users_created",
	})

	usersDisabled = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "within_website_x",
		Subsystem: "iamd",
		Name:      "users_disabled",
	})

	userErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "within_website_x",
		Subsystem: "iamd",
		Name:      "users_errors",
	}, []string{"call", "step"})
)

func New(dao *models.DAO) *Server {
	result := &Server{
		dao: dao,
	}

	return result
}

func (s *Server) CreateUser(ctx context.Context, req *iamv1.CreateUserReq) (*iamv1.CreateUserResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	u, err := s.dao.CreateUser(ctx, req.GetName())
	if err != nil {
		slog.ErrorContext(ctx, "can't create user", "err", err)
		userErrors.WithLabelValues("CreateUser", "create_user").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("can't create users")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	k, err := s.dao.CreateKey(ctx, u, "initial access key")
	if err != nil {
		slog.ErrorContext(ctx, "can't create user", "err", err)
		userErrors.WithLabelValues("CreateUser", "create_key").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("can't create initial access key")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	usersCreated.Inc()

	return &iamv1.CreateUserResp{
		User:            u.AsProto(),
		AccessKeyId:     k.AccessKeyID,
		SecretAccessKey: k.SecretAccessKey,
	}, nil
}

func (s *Server) DisableUser(ctx context.Context, req *iamv1.DisableUserReq) (*iamv1.DisableUserResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	err := s.dao.DisableUser(ctx, req.GetId(), req.GetReason())
	if err != nil {
		slog.ErrorContext(ctx, "can't create user", "err", err)
		userErrors.WithLabelValues("DisableUser", "upsert").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("user not found")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	return &iamv1.DisableUserResp{}, nil
}

func (s *Server) ListUsers(ctx context.Context, req *iamv1.ListUsersReq) (*iamv1.ListUsersResp, error) {
	users, err := s.dao.ListUsers(ctx, int(req.GetCount()), int(req.GetPage()))
	if err != nil {
		slog.ErrorContext(ctx, "can't create user", "err", err)
		userErrors.WithLabelValues("ListUsers", "select").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("user not found")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	var result []*iamv1.User
	for _, u := range users {
		result = append(result, u.AsProto())
	}

	return &iamv1.ListUsersResp{Users: result}, nil
}
