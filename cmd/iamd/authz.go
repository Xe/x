package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/proto"
	"within.website/x/cmd/iamd/models"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/iam/authz"
	"within.website/x/web/middleware/authctx"
)

// newAuthorizer builds the CEL evaluator over every mounted method. methodTypes
// maps a full method name to a sample request message so the CEL environment
// can resolve req fields.
func newAuthorizer() (*authz.Evaluator, error) {
	methodTypes := map[string]interface{}{
		authz.MethodName(iamv1.UserService_ServiceDesc.ServiceName, "CreateUser"):            &iamv1.CreateUserReq{},
		authz.MethodName(iamv1.UserService_ServiceDesc.ServiceName, "DisableUser"):           &iamv1.DisableUserReq{},
		authz.MethodName(iamv1.UserService_ServiceDesc.ServiceName, "ListUsers"):             &iamv1.ListUsersReq{},
		authz.MethodName(iamv1.KeyService_ServiceDesc.ServiceName, "CreateKey"):              &iamv1.CreateKeyReq{},
		authz.MethodName(iamv1.KeyService_ServiceDesc.ServiceName, "DisableKey"):             &iamv1.DisableKeyReq{},
		authz.MethodName(iamv1.KeyService_ServiceDesc.ServiceName, "ListKeys"):               &iamv1.ListKeysReq{},
		authz.MethodName(iamv1.PolicyService_ServiceDesc.ServiceName, "CreatePolicy"):        &iamv1.CreatePolicyReq{},
		authz.MethodName(iamv1.PolicyService_ServiceDesc.ServiceName, "DeletePolicy"):        &iamv1.DeletePolicyReq{},
		authz.MethodName(iamv1.PolicyService_ServiceDesc.ServiceName, "GetPolicy"):           &iamv1.GetPolicyReq{},
		authz.MethodName(iamv1.PolicyService_ServiceDesc.ServiceName, "ListPolicies"):        &iamv1.ListPoliciesReq{},
		authz.MethodName(iamv1.PolicyService_ServiceDesc.ServiceName, "AttachPolicy"):        &iamv1.AttachPolicyReq{},
		authz.MethodName(iamv1.PolicyService_ServiceDesc.ServiceName, "DetachPolicy"):        &iamv1.DetachPolicyReq{},
		authz.MethodName(iamv1.PolicyService_ServiceDesc.ServiceName, "ListPoliciesForUser"): &iamv1.ListPoliciesForUserReq{},
		authz.MethodName(stsv1.SigningKeyService_ServiceDesc.ServiceName, "GetSigningKey"):   &stsv1.GetSigningKeyRequest{},
		authz.MethodName(stsv1.SigningKeyService_ServiceDesc.ServiceName, "GetPublicKey"):    &stsv1.GetPublicKeyRequest{},
	}
	eval, err := authz.New(methodTypes)
	if err != nil {
		return nil, fmt.Errorf("can't build policy evaluator: %w", err)
	}
	return eval, nil
}

// fullServiceNames maps the short service names Twirp puts in ctx to the
// full proto names policy statements match against. Values come from the
// generated gRPC ServiceDescs, so they cannot drift from the protos.
var fullServiceNames = map[string]string{
	"UserService":       iamv1.UserService_ServiceDesc.ServiceName,
	"KeyService":        iamv1.KeyService_ServiceDesc.ServiceName,
	"PolicyService":     iamv1.PolicyService_ServiceDesc.ServiceName,
	"SigningKeyService": stsv1.SigningKeyService_ServiceDesc.ServiceName,
}

// policyInterceptor authorizes every Twirp call after the route middleware
// has authenticated the caller and resolved its user. It runs policy
// evaluation (see within.website/x/iam/authz) with the decoded request
// message; PolicyService handlers additionally gate on is_admin inside
// themselves, so administration can never be reached through a policy
// document. Policies are read per call so attaches and detaches take effect
// on the next request.
func policyInterceptor(lg *slog.Logger, dao *models.DAO, eval *authz.Evaluator) twirp.Interceptor {
	return func(next twirp.Method) twirp.Method {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			u, ok := authctx.User(ctx)
			if !ok {
				return nil, twirp.NewError(twirp.Unauthenticated, "no verified caller in context")
			}
			// Admin access must survive missing or corrupt policy documents so
			// the caller can repair policy attachments through PolicyService.
			if u.GetIsAdmin() {
				eval.Check(nil, "", nil, authz.PrincipalFromUser("", u))
				return next(ctx, req)
			}

			msg, ok := req.(proto.Message)
			if !ok {
				// Every mounted route takes a proto request; anything else is
				// a wiring fault. Fail closed.
				return nil, twirp.InternalErrorf("authz: request is not a proto message")
			}

			svc, _ := twirp.ServiceName(ctx)
			mth, _ := twirp.MethodName(ctx)
			method := authz.MethodName(fullServiceNames[svc], mth)
			keyID, _ := authctx.KeyID(ctx)
			principal := authz.PrincipalFromUser(keyID, u)

			policies, err := dao.PoliciesForUser(ctx, u.GetId())
			if err != nil {
				lg.ErrorContext(ctx, "authz: can't load policies", "err", err, "user", u.GetId())
				return nil, twirp.InternalErrorf("authz: can't load policies")
			}

			d := eval.Check(policies, method, msg, principal)
			if !d.Allowed {
				lg.WarnContext(ctx, "authz: denied", "method", method, "user", u.GetId(), "reason", d.Reason)
				return nil, twirp.NewError(twirp.PermissionDenied, d.Reason)
			}
			lg.DebugContext(ctx, "authz: allowed", "method", method, "user", u.GetId(), "reason", d.Reason)

			return next(ctx, req)
		}
	}
}
