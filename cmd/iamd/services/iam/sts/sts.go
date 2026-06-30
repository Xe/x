// Package sts implements the central STSService: it validates AWS SigV4-signed
// request material forwarded by a verifying service and returns the user that
// owns the signing key.
//
// This is the server side of web/middleware/sigv4/iamsts. A downstream service
// that must not hold signing secrets forwards the request material it actually
// received (method, path, query, host, headers) here; this handler reconstructs
// the canonical request, validates the signature with the local sigv4 verifier
// (without the body — the verifying service checks the body hash locally), and
// resolves the owning user. Signing secrets never leave iamd.
package sts

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"within.website/x/cmd/iamd/models"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
)

// amzTimeFormat is the AWS X-Amz-Date timestamp format.
const amzTimeFormat = "20060102T150405Z"

// Server implements stsv1.STSService.
type Server struct {
	dao      *models.DAO
	verifier *sigv4.Verifier
	stsv1.UnimplementedSTSServiceServer
}

// New returns an STS server that resolves secrets and users from dao and
// validates signatures with verifier (the same verifier iamd uses for its own
// routes).
func New(dao *models.DAO, verifier *sigv4.Verifier) *Server {
	return &Server{dao: dao, verifier: verifier}
}

// GetCallerIdentity validates forwarded SigV4 request material and returns the
// owning user. Verification failures are returned as twirp.Unauthenticated per
// the STS contract; missing forwarded fields are twirp.InvalidArgument; backing-
// store faults are twirp.Internal.
func (s *Server) GetCallerIdentity(ctx context.Context, req *stsv1.GetCallerIdentityReq) (*stsv1.GetCallerIdentityResp, error) {
	if req.GetMethod() == "" {
		return nil, twirp.InvalidArgumentError("method", "must be set")
	}
	if req.GetPath() == "" {
		return nil, twirp.InvalidArgumentError("path", "must be set")
	}
	if req.GetHost() == "" {
		return nil, twirp.InvalidArgumentError("host", "must be set")
	}

	// Rebuild an http.Header with canonical keys so the verifier's Get/Values
	// lookups (which canonicalize) find the forwarded values.
	headers := make(http.Header, len(req.GetHeaders()))
	for _, h := range req.GetHeaders() {
		headers.Add(h.GetName(), h.GetValue())
	}
	declared := headers.Get("X-Amz-Content-Sha256")
	if declared == "" {
		return nil, twirp.InvalidArgumentError("x-amz-content-sha256", "must be forwarded so the signature can be verified without the body")
	}

	keyID, err := s.verifier.VerifySignature(req.GetMethod(), req.GetPath(), req.GetQuery(), req.GetHost(), headers, declared)
	if err != nil {
		// Verification failures (bad signature, unknown or disabled key, clock
		// skew, scope mismatch, unsupported payload hash) are UNAUTHENTICATED.
		// Anything else is a server fault — e.g. the key store being down — and
		// must not be misreported as an auth denial.
		if isVerificationErr(err) {
			return nil, twirp.NewError(twirp.Unauthenticated, err.Error())
		}
		return nil, twirp.InternalErrorWith(err)
	}

	user, err := s.dao.GetUserByAccessKeyID(ctx, keyID)
	if err != nil {
		// A key that verified but maps to no enabled user is unauthenticated
		// rather than a server error, and must not leak that the key exists.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, twirp.NewError(twirp.Unauthenticated, "signing key has no enabled user")
		}
		return nil, twirp.InternalErrorWith(err)
	}

	return &stsv1.GetCallerIdentityResp{
		User:        user.AsProto(),
		AccessKeyId: keyID,
		SignedAt:    parseSignedAt(headers.Get("X-Amz-Date")),
	}, nil
}

// isVerificationErr reports whether err is one of the sigv4 package's
// verification-failure sentinels, as opposed to a backing-store fault or a
// missing Lookup (misconfiguration), which should surface as 500.
func isVerificationErr(err error) bool {
	switch {
	case errors.Is(err, sigv4.ErrMissingAuth),
		errors.Is(err, sigv4.ErrMissingSignedHost),
		errors.Is(err, sigv4.ErrUnknownKey),
		errors.Is(err, sigv4.ErrClockSkew),
		errors.Is(err, sigv4.ErrScopeMismatch),
		errors.Is(err, sigv4.ErrStreamingUnsupported),
		errors.Is(err, sigv4.ErrBodyHash),
		errors.Is(err, sigv4.ErrUnauthorized),
		errors.Is(err, sigv4.ErrBodyTooLarge):
		return true
	}
	return false
}

// parseSignedAt parses the forwarded X-Amz-Date into a timestamp for audit
// logging. It is best-effort: a missing or malformed date yields nil.
func parseSignedAt(amzDate string) *timestamppb.Timestamp {
	if amzDate == "" {
		return nil
	}
	t, err := time.Parse(amzTimeFormat, amzDate)
	if err != nil {
		return nil
	}
	return timestamppb.New(t)
}
