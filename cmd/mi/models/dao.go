package models

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"github.com/oklog/ulid/v2"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/gorm"
	gormPrometheus "gorm.io/plugin/prometheus"
)

var (
	ErrCantSwitchToYourself = errors.New("models: you can't switch to yourself")
)

type DAO struct {
	db *gorm.DB
}

func (d *DAO) DB() *gorm.DB {
	return d.db
}

func (d *DAO) Ping(ctx context.Context) error {
	if err := d.db.WithContext(ctx).Exec("select 1+1").Error; err != nil {
		return err
	}

	return nil
}

func New(dbLoc string) (*DAO, error) {
	db, err := gorm.Open(gormlite.Open(dbLoc), &gorm.Config{
		Logger: slogGorm.New(
			slogGorm.WithErrorField("err"),
			slogGorm.WithRecordNotFoundError(),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(
		&Member{},
		&Switch{},
		&Blogpost{},
		&Event{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	db.Use(gormPrometheus.New(gormPrometheus.Config{
		DBName: "mi",
	}))

	return &DAO{db: db}, nil
}

func (d *DAO) Members(ctx context.Context) ([]Member, error) {
	var result []Member
	if err := d.db.WithContext(ctx).Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (d *DAO) WhoIsFront(ctx context.Context) (*Switch, error) {
	var sw Switch
	if err := d.db.Joins("Member").Order("created_at DESC").First(&sw).Error; err != nil {
		return nil, err
	}

	return &sw, nil
}

func (d *DAO) SwitchFront(ctx context.Context, memberName string) (*Switch, *Switch, error) {
	var old Switch
	tx := d.db.Begin()

	if err := tx.WithContext(ctx).Joins("Member").Where("ended_at IS NULL").First(&old).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	if old.Member.Name == memberName {
		tx.WithContext(ctx).Rollback()
		return nil, nil, ErrCantSwitchToYourself
	}

	now := time.Now()
	old.EndedAt = &now
	if err := tx.WithContext(ctx).Save(&old).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	var newMember Member
	if err := tx.WithContext(ctx).Where("name = ?", memberName).First(&newMember).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	new := Switch{
		ID:       ulid.MustNew(ulid.Now(), rand.Reader).String(),
		MemberID: newMember.ID,
	}

	if err := tx.WithContext(ctx).Create(&new).Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	if err := tx.WithContext(ctx).Commit().Error; err != nil {
		tx.Rollback()
		return nil, nil, err
	}

	return &old, &new, nil
}

func (d *DAO) GetSwitch(ctx context.Context, id string) (*Switch, error) {
	var sw Switch
	if err := d.db.WithContext(ctx).
		Joins("Member").
		Where("switches.id = ?", id).
		First(&sw).Error; err != nil {
		return nil, err
	}

	return &sw, nil
}

func (d *DAO) ListSwitches(ctx context.Context, count, page int) ([]Switch, error) {
	var switches []Switch
	if err := d.db.WithContext(ctx).
		Joins("Member").
		Order("created_at DESC").
		Limit(count).
		Offset(count * page).
		Find(&switches).Error; err != nil {
		return nil, err
	}

	return switches, nil
}
