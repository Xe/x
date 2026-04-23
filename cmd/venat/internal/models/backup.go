package models

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ncruces/go-sqlite3"
)

func (d *DAO) Backup() {
	slog.Info("starting backup")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err := d.backup(ctx, d.backupDBLoc)
	if err != nil {
		slog.Error("failed to backup database", "err", err)
	}
	slog.Info("backup done")
}

func (d *DAO) backup(ctx context.Context, to string) error {
	db, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	defer conn.Close()

	if err := conn.Raw(func(dca any) error {
		conn, ok := dca.(sqlite3.DriverConn)
		if !ok {
			return fmt.Errorf("db connection is not a sqlite3 connection, it is %T", dca)
		}

		bu, err := conn.Raw().BackupInit("main", to)
		if err != nil {
			return fmt.Errorf("failed to initialize backup: %w", err)
		}
		defer bu.Close()

		var done bool
		for !done {
			done, err = bu.Step(bu.Remaining())
			if err != nil {
				return fmt.Errorf("failed to backup database: %w", err)
			}
		}

		if err := bu.Close(); err != nil {
			return fmt.Errorf("failed to close backup: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	return nil
}
