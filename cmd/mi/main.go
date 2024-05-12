package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	slogGorm "github.com/orandin/slog-gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"within.website/x/cmd/mi/models"
	"within.website/x/internal"
	pb "within.website/x/proto/mi"
)

var (
	bind  = flag.String("bind", ":8080", "HTTP bind address")
	dbLoc = flag.String("db-loc", "./var/data.db", "")
)

func main() {
	internal.HandleStartup()

	db, err := gorm.Open(sqlite.Open(*dbLoc), &gorm.Config{
		Logger: slogGorm.New(
			slogGorm.WithErrorField("err"),
			slogGorm.WithRecordNotFoundError(),
		),
	})
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}

	if err := db.AutoMigrate(&models.Member{}, &models.Switch{}); err != nil {
		slog.Error("failed to migrate schema", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.Handle(pb.SwitchTrackerPathPrefix, pb.NewSwitchTrackerServer(NewSwitchTracker(db)))

	i := &Importer{db: db}
	i.Mount(mux)

	slog.Info("starting server", "bind", *bind)
	slog.Error("server stopped", "err", http.ListenAndServe(*bind, mux))
}
