package main

import (
	"flag"
	"fmt"
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
	bind         = flag.String("bind", ":8080", "HTTP bind address")
	dbLoc        = flag.String("db-loc", "./var/data.db", "")
	internalBind = flag.String("internal-bind", ":9195", "HTTP internal routes bind address")
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

	dao := models.New(db)

	mux := http.NewServeMux()

	mux.Handle(pb.SwitchTrackerPathPrefix, pb.NewSwitchTrackerServer(NewSwitchTracker(db)))
	mux.Handle("/front", &HomeFrontShim{dao: dao})

	i := &Importer{db: db}
	i.Mount(http.DefaultServeMux)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Exec("select 1+1").Error; err != nil {
			http.Error(w, "database not healthy", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	go func() {
		slog.Info("starting internal server", "bind", *internalBind)
		slog.Error("internal server stopped", "err", http.ListenAndServe(*internalBind, nil))
	}()

	slog.Info("starting server", "bind", *bind)
	slog.Error("server stopped", "err", http.ListenAndServe(*bind, mux))
}
