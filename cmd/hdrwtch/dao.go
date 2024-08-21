package main

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormPrometheus "gorm.io/plugin/prometheus"
)

type DAO struct {
	db *gorm.DB
}

func (dao *DAO) DB() *gorm.DB {
	return dao.db
}

func New(dbLoc string) (*DAO, error) {
	db, err := gorm.Open(gormlite.Open(dbLoc), &gorm.Config{
		Logger: slogGorm.New(
			slogGorm.WithErrorField("err"),
			slogGorm.WithRecordNotFoundError(),
			slogGorm.SetLogLevel(slogGorm.DefaultLogType, slog.LevelInfo),
			//slogGorm.WithTraceAll(),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(
		&TelegramUser{},
		&Probe{},
		&ProbeResult{},
		&Doc{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	db.Use(gormPrometheus.New(gormPrometheus.Config{
		DBName: "hdrwtch",
	}))

	return &DAO{db: db}, nil
}

func (dao *DAO) GetUser(ctx context.Context, id int64) (*TelegramUser, error) {
	var user TelegramUser
	if err := dao.db.First(&user, id).WithContext(ctx).Error; err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (dao *DAO) UpsertUser(ctx context.Context, user *TelegramUser) error {
	if err := dao.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},                                                           // primary key
		DoUpdates: clause.AssignmentColumns([]string{"first_name", "last_name", "photo_url", "auth_date"}), // column needed to be updated
	}).
		Create(user).
		WithContext(ctx).
		Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (dao *DAO) CreateProbe(ctx context.Context, probe *Probe, tu *TelegramUser) error {
	// Check if user has reached probe limit
	var count int64
	if err := dao.db.Model(&Probe{}).Where("user_id = ?", tu.ID).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count probes: %w", err)
	}

	if count >= int64(tu.ProbeLimit) {
		return fmt.Errorf("probe limit reached")
	}

	if err := dao.db.Create(probe).WithContext(ctx).Error; err != nil {
		return fmt.Errorf("failed to create probe: %w", err)
	}

	return nil
}

func (dao *DAO) CountProbes(ctx context.Context, userID int64) (int64, error) {
	var count int64
	if err := dao.db.Model(&Probe{}).Where("user_id = ?", userID).Count(&count).WithContext(ctx).Error; err != nil {
		return 0, fmt.Errorf("failed to count probes: %w", err)
	}

	return count, nil
}

func (dao *DAO) GetProbe(ctx context.Context, id string, userID int64) (*Probe, error) {
	var probe Probe

	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid probe ID: %w", err)
	}

	if err := dao.db.Preload("LastResult").First(&probe, idInt).WithContext(ctx).Error; err != nil {
		return nil, fmt.Errorf("failed to get probe: %w", err)
	}

	if probe.UserID != userID {
		return nil, fmt.Errorf("probe does not belong to user")
	}

	return &probe, nil
}

func (dao *DAO) CreateProbeResult(ctx context.Context, tx *gorm.DB, probe Probe, result *ProbeResult) error {
	if err := tx.Create(result).WithContext(ctx).Error; err != nil {
		return fmt.Errorf("failed to create probe result: %w", err)
	}

	probe.LastResultID = result.ID
	if err := tx.Save(&probe).WithContext(ctx).Error; err != nil {
		return fmt.Errorf("failed to update probe: %w", err)
	}

	return nil
}

func (dao *DAO) GetDoc(slug string) (*Doc, error) {
	var doc Doc
	if err := dao.db.Where("slug = ?", slug).First(&doc).Error; err != nil {
		return nil, err
	}

	return &doc, nil
}
