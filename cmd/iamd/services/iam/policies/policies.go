package policies

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
	"within.website/x/iam/authz"
	"within.website/x/web/middleware/authctx"
)

type Server struct {
	dao *models.DAO

	iamv1.UnimplementedPolicyServiceServer
}

var (
	policiesCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "iam_policy_documents_created_total",
		Help: "Number of policy documents created.",
	})

	policiesDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "iam_policy_documents_deleted_total",
		Help: "Number of policy documents deleted.",
	})

	policiesAttached = promauto.NewCounter(prometheus.CounterOpts{
		Name: "iam_policy_attachments_created_total",
		Help: "Number of user-policy attachments created.",
	})

	policiesDetached = promauto.NewCounter(prometheus.CounterOpts{
		Name: "iam_policy_attachments_removed_total",
		Help: "Number of user-policy attachments removed.",
	})

	policyErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "iam_policy_service_errors_total",
		Help: "Errors in policy service calls, by call and failing step.",
	}, []string{"call", "step"})
)

func New(dao *models.DAO) *Server {
	return &Server{dao: dao}
}

// adminCaller resolves the authenticated caller and requires is_admin.
// Policy administration is gated by the admin flag, never by policies, so
// no policy document can lock an admin out of the management surface.
func adminCaller(ctx context.Context) (*iamv1.User, error) {
	u, ok := authctx.User(ctx)
	if !ok {
		return nil, twirp.NewError(twirp.Unauthenticated, "no verified caller in context")
	}
	if !u.GetIsAdmin() {
		return nil, twirp.NewError(twirp.PermissionDenied, "policy administration requires an admin caller")
	}
	return u, nil
}

func (s *Server) CreatePolicy(ctx context.Context, req *iamv1.CreatePolicyReq) (*iamv1.CreatePolicyResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if _, err := adminCaller(ctx); err != nil {
		return nil, err
	}

	doc := req.GetPolicy()
	if doc == nil || doc.GetName() == "" {
		return nil, twirp.InvalidArgumentError("policy.name", "must be set")
	}
	if len(doc.GetStatements()) == 0 {
		return nil, twirp.InvalidArgumentError("policy.statements", "at least one statement is required")
	}
	for _, st := range doc.GetStatements() {
		if st.GetEffect() == iamv1.Effect_EFFECT_UNSPECIFIED {
			return nil, twirp.InvalidArgumentError("policy.statements", "statement {effect} must be ALLOW or DENY")
		}
		if len(st.GetMethods()) == 0 {
			return nil, twirp.InvalidArgumentError("policy.statements", "statement must list at least one method pattern")
		}
		// Syntax-check filters at write time so typos are rejected here
		// instead of failing closed at evaluation. Full type-checking needs
		// the owning service's request types and still happens there.
		if err := authz.TryCompile(st.GetFilter()); err != nil {
			return nil, twirp.InvalidArgumentError("policy.statements", err.Error())
		}
	}

	p, err := s.dao.CreatePolicy(ctx, doc.GetName(), doc.GetStatements())
	if err != nil {
		slog.ErrorContext(ctx, "can't create policy", "err", err)
		policyErrors.WithLabelValues("CreatePolicy", "create").Inc()
		switch {
		case errors.Is(err, models.ErrDuplicatePolicyName):
			return nil, twirp.NewError(twirp.AlreadyExists, "policy name already in use")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	policiesCreated.Inc()

	return &iamv1.CreatePolicyResp{Policy: p}, nil
}

func (s *Server) GetPolicy(ctx context.Context, req *iamv1.GetPolicyReq) (*iamv1.GetPolicyResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if _, err := adminCaller(ctx); err != nil {
		return nil, err
	}

	p, err := s.dao.GetPolicy(ctx, req.GetPolicyId())
	if err != nil {
		slog.ErrorContext(ctx, "can't get policy", "err", err)
		policyErrors.WithLabelValues("GetPolicy", "select").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("policy not found")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	return &iamv1.GetPolicyResp{Policy: p}, nil
}

func (s *Server) ListPolicies(ctx context.Context, req *iamv1.ListPoliciesReq) (*iamv1.ListPoliciesResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if _, err := adminCaller(ctx); err != nil {
		return nil, err
	}

	ps, err := s.dao.ListPolicies(ctx, int(req.GetCount()), int(req.GetPage()))
	if err != nil {
		slog.ErrorContext(ctx, "can't list policies", "err", err)
		policyErrors.WithLabelValues("ListPolicies", "select").Inc()
		return nil, twirp.InternalErrorWith(err)
	}

	return &iamv1.ListPoliciesResp{Policies: ps}, nil
}

func (s *Server) DeletePolicy(ctx context.Context, req *iamv1.DeletePolicyReq) (*iamv1.DeletePolicyResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if _, err := adminCaller(ctx); err != nil {
		return nil, err
	}

	if err := s.dao.DeletePolicy(ctx, req.GetPolicyId()); err != nil {
		slog.ErrorContext(ctx, "can't delete policy", "err", err)
		policyErrors.WithLabelValues("DeletePolicy", "delete").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("policy not found")
		case errors.Is(err, models.ErrPolicyAttached):
			return nil, twirp.NewError(twirp.FailedPrecondition, "policy is still attached to at least one user")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	policiesDeleted.Inc()

	return &iamv1.DeletePolicyResp{}, nil
}

func (s *Server) AttachPolicy(ctx context.Context, req *iamv1.AttachPolicyReq) (*iamv1.AttachPolicyResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if _, err := adminCaller(ctx); err != nil {
		return nil, err
	}

	if err := s.dao.AttachPolicy(ctx, req.GetPolicyId(), req.GetUserId()); err != nil {
		slog.ErrorContext(ctx, "can't attach policy", "err", err)
		policyErrors.WithLabelValues("AttachPolicy", "attach").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("policy or user not found")
		case errors.Is(err, models.ErrAlreadyAttached):
			return nil, twirp.NewError(twirp.AlreadyExists, "policy already attached to user")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	policiesAttached.Inc()

	return &iamv1.AttachPolicyResp{}, nil
}

func (s *Server) DetachPolicy(ctx context.Context, req *iamv1.DetachPolicyReq) (*iamv1.DetachPolicyResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if _, err := adminCaller(ctx); err != nil {
		return nil, err
	}

	if err := s.dao.DetachPolicy(ctx, req.GetPolicyId(), req.GetUserId()); err != nil {
		slog.ErrorContext(ctx, "can't detach policy", "err", err)
		policyErrors.WithLabelValues("DetachPolicy", "detach").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("no such attachment")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	policiesDetached.Inc()

	return &iamv1.DetachPolicyResp{}, nil
}

func (s *Server) ListPoliciesForUser(ctx context.Context, req *iamv1.ListPoliciesForUserReq) (*iamv1.ListPoliciesForUserResp, error) {
	if err := protovalidate.Validate(req); err != nil {
		return nil, twirp.NewError(twirp.InvalidArgument, err.Error())
	}
	if _, err := adminCaller(ctx); err != nil {
		return nil, err
	}

	ps, err := s.dao.PoliciesForUser(ctx, req.GetUserId())
	if err != nil {
		slog.ErrorContext(ctx, "can't list policies for user", "err", err)
		policyErrors.WithLabelValues("ListPoliciesForUser", "select").Inc()
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, twirp.NotFoundError("user not found")
		default:
			return nil, twirp.InternalErrorWith(err)
		}
	}

	return &iamv1.ListPoliciesForUserResp{Policies: ps}, nil
}
