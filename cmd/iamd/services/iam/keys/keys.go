package keys

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
	"within.website/x/web/middleware/sigv4"
)

type Server struct {
	dao *models.DAO

	iamv1.UnimplementedKeyServiceServer
}

var (
	keysCreated = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "within_website_x",
		Subsystem: "iamd",
		Name:      "keys_created",
	})

	keysDisabled = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "within_website_x",
		Subsystem: "iamd",
		Name:      "keys_disabled",
	})

	keyErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "within_website_x",
		Subsystem: "iamd",
		Name:      "keys_errors",
	}, []string{"call", "step"})
)

func New(dao *models.DAO) *Server {
	return &Server{
		dao: dao,
	}
}

// caller resolves the authenticated caller from the request context. Every Key
// route runs after the UserMiddleware that populates it, so a missing caller is
// reported as Unauthenticated rather than a server fault.
func caller(ctx context.Context) (*iamv1.User, error) {
	u, ok := sigv4.User(ctx)
	if !ok {
		return nil, twirp.NewError(twirp.Unauthenticated, "no authenticated caller")
	}
	return u, nil
}

func (s *Server) CreateKey(ctx context.Context, req *iamv1.CreateKeyReq) (*iamv1.CreateKeyResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	c, err := caller(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetUserId() == "" {
		return nil, twirp.InvalidArgumentError("user_id", "must be set to create a key")
	}

	// A non-admin may only provision keys for itself; an admin may target any user.
	if !c.GetIsAdmin() && req.GetUserId() != c.GetId() {
		return nil, twirp.NewError(twirp.PermissionDenied, "can only create keys for your own user")
	}

	u, err := s.dao.GetUser(ctx, req.GetUserId())
	if err != nil {
		slog.Error("can't create key", "err", err)
		keyErrors.WithLabelValues("CreateKey", "get_user").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("user not found")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	k, err := s.dao.CreateKey(ctx, u, req.GetComment())
	if err != nil {
		slog.Error("can't create key", "err", err)
		keyErrors.WithLabelValues("CreateKey", "create_key").Inc()
		return nil, twirp.InternalErrorWith(err)
	}

	keysCreated.Inc()

	return &iamv1.CreateKeyResp{
		Key:             k.AsProto(),
		SecretAccessKey: k.SecretAccessKey,
	}, nil
}

func (s *Server) DisableKey(ctx context.Context, req *iamv1.DisableKeyReq) (*iamv1.DisableKeyResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}

	c, err := caller(ctx)
	if err != nil {
		return nil, err
	}

	// An admin may disable any key (an explicit user_id scopes the disable); a
	// non-admin is pinned to their own user, so the DAO rejects another user's
	// key as not found without leaking that it exists.
	scopeUserID := req.GetUserId()
	if !c.GetIsAdmin() {
		scopeUserID = c.GetId()
	}

	if err := s.dao.DisableKey(ctx, req.GetKeyId(), req.GetReason(), scopeUserID); err != nil {
		slog.Error("can't disable key", "err", err)
		keyErrors.WithLabelValues("DisableKey", "disable").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("key not found")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	keysDisabled.Inc()

	return &iamv1.DisableKeyResp{}, nil
}

func (s *Server) ListKeys(ctx context.Context, req *iamv1.ListKeysReq) (*iamv1.ListKeysResp, error) {
	// Intentionally not protovalidate-d: page is 0-based at the DAO
	// (Offset(count*page), so page=0 is the first page), yet the proto marks it
	// required (non-zero) — validating would reject the first page and make
	// page=1 skip the first count rows. UserService.ListUsers skips validation
	// for the same reason.
	c, err := caller(ctx)
	if err != nil {
		return nil, err
	}

	// An admin sees keys across all users (an explicit user_id scopes the view);
	// a non-admin may only list their own keys, and asking for another user's is
	// rejected outright rather than silently clamped.
	var userID string
	switch {
	case c.GetIsAdmin():
		userID = req.GetUserId()
	default:
		if req.GetUserId() != "" && req.GetUserId() != c.GetId() {
			return nil, twirp.NewError(twirp.PermissionDenied, "cannot list keys for another user")
		}
		userID = c.GetId()
	}

	keys, err := s.dao.ListKeys(ctx, int(req.GetCount()), int(req.GetPage()), userID)
	if err != nil {
		slog.Error("can't list keys", "err", err)
		keyErrors.WithLabelValues("ListKeys", "select").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("user not found")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	var result []*iamv1.Key
	for _, k := range keys {
		result = append(result, k.AsProto())
	}

	return &iamv1.ListKeysResp{Keys: result}, nil
}
