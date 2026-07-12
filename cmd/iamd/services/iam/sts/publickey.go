package sts

import (
	"context"
	"crypto/x509"
	"errors"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4a"
)

// GetPublicKey resolves the access key and its owning user, derives the
// SigV4A keypair from the stored secret, and returns the public half in PKIX
// DER form. Unlike GetSigningKey there is no scope validation: the keypair
// is a pure function of the credential, not of a (date, region, service)
// tuple, and the public key is not sensitive material.
func (s *SigningKeys) GetPublicKey(ctx context.Context, req *stsv1.GetPublicKeyRequest) (*stsv1.GetPublicKeyResponse, error) {
	if req.GetAccessKeyId() == "" {
		return nil, twirp.RequiredArgumentError("access_key_id")
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

	priv, err := sigv4a.DeriveKeyPair(k.AccessKeyID, k.SecretAccessKey)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &stsv1.GetPublicKeyResponse{
		PublicKey: der,
		Identity: &stsv1.TokenIdentity{
			AccessKeyId: k.AccessKeyID,
			PrincipalId: k.User.UUID,
			DisplayName: k.User.Name,
		},
		CacheUntil: timestamppb.New(s.now().Add(s.cacheTTL)),
	}, nil
}
