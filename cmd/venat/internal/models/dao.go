package models

import (
	"context"
	"fmt"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/gorm"
	gormPrometheus "gorm.io/plugin/prometheus"
)

type DAO struct {
	db          *gorm.DB
	backupDBLoc string
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

func New(dbLoc, backupDBLoc string) (*DAO, error) {
	db, err := gorm.Open(gormlite.Open(dbLoc), &gorm.Config{
		Logger: slogGorm.New(
			slogGorm.WithErrorField("err"),
			slogGorm.WithRecordNotFoundError(),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	db.Use(gormPrometheus.New(gormPrometheus.Config{
		DBName: "venat",
	}))

	return &DAO{db: db, backupDBLoc: backupDBLoc}, nil
}
