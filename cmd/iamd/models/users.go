package models

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	iamv1 "within.website/x/gen/within/website/x/iam/v1"
)

type User struct {
	gorm.Model // adds CreatedAt, UpdatedAt, DeletedAt

	UUID          string `gorm:"uniqueIndex"` // uuidv7
	Name          string // human-readable name
	DisableReason *string
	IsAdmin       bool
}

func (u *User) AsProto() *iamv1.User {
	var dr string

	if u.DisableReason != nil {
		dr = *u.DisableReason
	}

	return &iamv1.User{
		Id:            u.UUID,
		Name:          u.Name,
		CreatedAt:     timestamppb.New(u.CreatedAt),
		UpdatedAt:     timestamppb.New(u.UpdatedAt),
		DisabledAt:    timestamppb.New(u.DeletedAt.Time),
		DisableReason: dr,
		IsAdmin:       u.IsAdmin,
	}
}

func (d *DAO) CreateUser(ctx context.Context, name string) (*User, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	u := User{
		UUID: id.String(),
		Name: name,
	}

	if err := d.users.Create(ctx, &u); err != nil {
		return nil, err
	}

	return &u, nil
}

func (d *DAO) DisableUser(ctx context.Context, userID, reason string) error {
	keys := gorm.G[Key](d.db.Unscoped())
	users := gorm.G[User](d.db.Unscoped())

	u, err := users.Where("uuid = ?", userID).First(ctx)
	if err != nil {
		return err
	}

	if _, err := users.Where("uuid = ?", userID).Update(ctx, "disable_reason", reason); err != nil {
		return err
	}

	if _, err := keys.Where("user_id = ?", u.Model.ID).Update(ctx, "disable_reason", "user disabled: "+reason); err != nil {
		return err
	}

	if _, err := keys.Where("user_id = ?", u.Model.ID).Delete(ctx); err != nil {
		return err
	}

	if _, err := users.Where("uuid = ?", userID).Delete(ctx); err != nil {
		return err
	}

	return nil
}

func (d *DAO) ListUsers(ctx context.Context, count, page int) ([]User, error) {
	users, err := d.users.
		Order("created_at DESC").
		Limit(count).
		Offset(count * page).
		Find(ctx)

	if err != nil {
		return nil, err
	}

	return users, nil
}

// GetUser returns the user with the given public id (UUID). It returns an error
// wrapping gorm.ErrRecordNotFound if no such user exists (or the user is
// soft-deleted).
func (d *DAO) GetUser(ctx context.Context, userID string) (*User, error) {
	u, err := d.users.Where("uuid = ?", userID).First(ctx)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByAccessKeyID resolves an access key id to its owning user. The
// accessKeyID is the value the sigv4 middleware stores in the request context
// (see web/middleware/sigv4.KeyID). It returns an error wrapping
// gorm.ErrRecordNotFound if the key or its user does not exist, including when
// either has been soft-deleted (disabled).
func (d *DAO) GetUserByAccessKeyID(ctx context.Context, accessKeyID string) (*User, error) {
	k, err := d.keys.Where("access_key_id = ?", accessKeyID).First(ctx)
	if err != nil {
		return nil, err
	}
	u, err := d.users.Where("id = ?", k.UserID).First(ctx)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
