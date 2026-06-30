package models

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
	"within.website/x/web/middleware/sigv4/sigv4keygen"
)

type Key struct {
	gorm.Model // adds CreatedAt, UpdatedAt, DeletedAt

	AccessKeyID     string `gorm:"uniqueIndex"` // randomly generated
	SecretAccessKey string
	Comment         string
	DisableReason   *string
	UserID          uint  // User that owns the key
	User            *User `gorm:"foreignKey:UserID"`
}

func (k *Key) AsProto() *iamv1.Key {
	var dr string

	if k.DisableReason != nil {
		dr = *k.DisableReason
	}

	return &iamv1.Key{
		AccessKeyId:   k.AccessKeyID,
		Comment:       k.Comment,
		CreatedAt:     timestamppb.New(k.CreatedAt),
		UpdatedAt:     timestamppb.New(k.UpdatedAt),
		DisabledAt:    timestamppb.New(k.DeletedAt.Time),
		DisableReason: dr,
	}
}

func (d *DAO) CreateKey(ctx context.Context, user *User, comment string) (*Key, error) {
	ak, sk := sigv4keygen.Next()

	k := Key{
		AccessKeyID:     ak,
		SecretAccessKey: sk,
		Comment:         comment,
		UserID:          user.Model.ID,
	}

	if err := d.keys.Create(ctx, &k); err != nil {
		return nil, err
	}

	return &k, nil
}

// DisableKey soft-deletes the key with the given access key id. When userID is
// non-empty, the key must belong to that user or gorm.ErrRecordNotFound is
// returned without disabling anything — this lets a caller scope the operation
// so it cannot revoke another user's key. An empty userID skips the ownership
// check (administrative override).
func (d *DAO) DisableKey(ctx context.Context, keyID, reason, userID string) error {
	k, err := d.keys.Where("access_key_id = ?", keyID).First(ctx)
	if err != nil {
		return err
	}

	if userID != "" {
		u, err := d.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		if k.UserID != u.Model.ID {
			return gorm.ErrRecordNotFound
		}
	}

	if _, err := d.keys.Where("access_key_id = ?", keyID).Update(ctx, "disable_reason", reason); err != nil {
		return err
	}

	if _, err := d.keys.Where("access_key_id = ?", keyID).Delete(ctx); err != nil {
		return err
	}

	return nil
}

func (d *DAO) ListKeys(ctx context.Context, count, page int, userID string) ([]Key, error) {
	if userID == "" {
		return d.keys.
			Order("created_at DESC").
			Limit(count).
			Offset(count * page).
			Find(ctx)
	}

	u, err := d.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	return d.keys.
		Where("user_id = ?", u.Model.ID).
		Order("created_at DESC").
		Limit(count).
		Offset(count * page).
		Find(ctx)
}

// SecretFor returns the secret access key for the given access key id. It is the
// lookup the sigv4 Verifier uses to recompute signatures. A disabled (soft-
// deleted) key is excluded by GORM's default scope, so it surfaces as
// gorm.ErrRecordNotFound — which callers map to "unknown key".
func (d *DAO) SecretFor(ctx context.Context, accessKeyID string) (string, error) {
	k, err := d.keys.Where("access_key_id = ?", accessKeyID).First(ctx)
	if err != nil {
		return "", err
	}
	return k.SecretAccessKey, nil
}
