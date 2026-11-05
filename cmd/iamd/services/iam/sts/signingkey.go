// Package sts hosts iamd's security-token-service surface: the
// SigningKeyService that distributes SigV4 derived signing keys to
// downstream verifiers.
package sts

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"within.website/x/cmd/iamd/models"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/sigv4"
)

// maxClockSkew mirrors the verifier's clock-skew window: a request signed at
// 23:59:59 with scope date D is legitimately verifiable until 00:15 on D+1,
// so a key for D must outlive its UTC day by exactly this much.
const maxClockSkew = 15 * time.Minute

var dateRe = regexp.MustCompile(`^[0-9]{8}$`)

// SigningKeys implements stsv1.SigningKeyService: it derives per-scope SigV4
// signing keys from stored secrets so downstream services can verify request
// signatures locally. The raw secret never leaves this process; the derived
// key is bounded to one (access key, UTC day, region, service) scope.
type SigningKeys struct {
	dao      *models.DAO
	cacheTTL time.Duration

	// Now is overridable for tests. Defaults to time.Now.
	Now func() time.Time

	stsv1.UnimplementedSigningKeyServiceServer
}

// NewSigningKeys returns a SigningKeys server that issues keys only for the
// fleet-wide (region, service) scope and advises callers to re-fetch every
// cacheTTL, which bounds how long a disabled key keeps verifying downstream.
func NewSigningKeys(dao *models.DAO, cacheTTL time.Duration) *SigningKeys {
	return &SigningKeys{dao: dao, cacheTTL: cacheTTL}
}

// GetSigningKey validates the requested scope, resolves the key and its
// owning user, and returns the derived signing key with caching bounds.
// Neither the secret nor the derived key is ever logged.
func (s *SigningKeys) GetSigningKey(ctx context.Context, req *stsv1.GetSigningKeyRequest) (*stsv1.GetSigningKeyResponse, error) {
	if req.GetAccessKeyId() == "" {
		return nil, twirp.RequiredArgumentError("access_key_id")
	}
	if req.GetRegion() == "" {
		return nil, twirp.RequiredArgumentError("region")
	}
	if req.GetService() == "" {
		return nil, twirp.RequiredArgumentError("service")
	}
	if !dateRe.MatchString(req.GetDate()) {
		return nil, twirp.InvalidArgumentError("date", "must be YYYYMMDD")
	}
	day, err := time.Parse("20060102", req.GetDate())
	if err != nil {
		return nil, twirp.InvalidArgumentError("date", "must be a real UTC date in YYYYMMDD form")
	}

	now := s.now()
	today := now.UTC().Truncate(24 * time.Hour)
	if d := day.Sub(today); d > 24*time.Hour || d < -24*time.Hour {
		return nil, twirp.NewError(twirp.PermissionDenied, "signing keys are not issued for this date")
	}

	k, err := s.dao.GetKeyWithUser(ctx, req.GetAccessKeyId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, twirp.NotFoundError("unknown access key id")
		}
		return nil, twirp.InternalErrorWith(err)
	}
	// Disabled credentials are PERMISSION_DENIED, distinct from NOT_FOUND, so
	// the verifier can log the difference; its client-facing error is the
	// same for both.
	if k.DeletedAt.Valid {
		return nil, twirp.NewError(twirp.PermissionDenied, "access key is disabled")
	}
	if k.User == nil || k.User.DeletedAt.Valid {
		return nil, twirp.NewError(twirp.PermissionDenied, "owning user is disabled")
	}

	notValidAfter := day.AddDate(0, 0, 1).Add(maxClockSkew)
	cacheUntil := now.Add(s.cacheTTL)
	if cacheUntil.After(notValidAfter) {
		cacheUntil = notValidAfter
	}

	id, policies, err := s.identityAndPolicies(ctx, k)
	if err != nil {
		return nil, twirp.InternalErrorf("can't load identity policies")
	}

	return &stsv1.GetSigningKeyResponse{
		SigningKey:    sigv4.DeriveSigningKey(k.SecretAccessKey, req.GetDate(), req.GetRegion(), req.GetService()),
		Identity:      id,
		Policies:      policies,
		NotValidAfter: timestamppb.New(notValidAfter),
		CacheUntil:    timestamppb.New(cacheUntil),
	}, nil
}

// identityAndPolicies builds the TokenIdentity for the key's owning user and
// that user's effective policy set. Admin identities carry an empty policy
// list: they bypass evaluation, so there is nothing for a verifier to fetch.
// The key is pre-checked (user present, not disabled) by both callers.
func (s *SigningKeys) identityAndPolicies(ctx context.Context, k *models.Key) (*stsv1.TokenIdentity, []*iamv1.Policy, error) {
	id := &stsv1.TokenIdentity{
		AccessKeyId: k.AccessKeyID,
		PrincipalId: k.User.UUID,
		DisplayName: k.User.Name,
		IsAdmin:     k.User.IsAdmin,
	}
	if k.User.IsAdmin {
		return id, nil, nil
	}
	ps, err := s.dao.PoliciesForUser(ctx, k.User.UUID)
	if err != nil {
		return nil, nil, err
	}
	return id, ps, nil
}

func (s *SigningKeys) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
