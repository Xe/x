package main

import (
	"context"
	"fmt"
	"strconv"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/gorm"
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
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(
		&TelegramUser{},
		&Probe{},
		&ProbeResult{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	db.Use(gormPrometheus.New(gormPrometheus.Config{
		DBName: "hdrwtch",
	}))

	return &DAO{db: db}, nil
}

func (dao *DAO) GetProbe(ctx context.Context, id string, userID string) (*Probe, error) {
	var probe Probe

	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid probe ID: %w", err)
	}

	if err := dao.db.First(&probe, idInt).WithContext(ctx).Error; err != nil {
		return nil, fmt.Errorf("failed to get probe: %w", err)
	}

	if probe.UserID != userID {
		return nil, fmt.Errorf("probe does not belong to user")
	}

	return &probe, nil
}
