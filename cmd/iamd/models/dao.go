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
	db    *gorm.DB
	keys  gorm.Interface[Key]
	users gorm.Interface[User]
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
	dao, err := Open(dbLoc)
	if err != nil {
		return nil, err
	}

	if err := dao.db.Use(gormPrometheus.New(gormPrometheus.Config{
		DBName: "iamd",
	})); err != nil {
		return nil, fmt.Errorf("failed to enable prometheus metrics: %w", err)
	}

	return dao, nil
}

// Open connects to the SQLite database at dbLoc, runs migrations, and returns a
// ready DAO. Unlike New, it does not register Prometheus collectors, so it is
// safe to call repeatedly (for example from tests, where New's global collector
// would collide) or for embedded use without metrics.
func Open(dbLoc string) (*DAO, error) {
	db, err := gorm.Open(gormlite.Open(dbLoc), &gorm.Config{
		Logger: slogGorm.New(
			slogGorm.WithErrorField("err"),
			slogGorm.WithRecordNotFoundError(),
		),
		PropagateUnscoped: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(
		&User{},
		&Key{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	return &DAO{
		db:    db,
		keys:  gorm.G[Key](db),
		users: gorm.G[User](db),
	}, nil
}
