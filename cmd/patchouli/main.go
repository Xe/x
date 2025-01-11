package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"path/filepath"

	"connectrpc.com/connect"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	slogGorm "github.com/orandin/slog-gorm"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"
	gormPrometheus "gorm.io/plugin/prometheus"
	"within.website/x/buf/patchouli"
	"within.website/x/buf/patchouli/patchouliconnect"
	"within.website/x/cmd/patchouli/ytdlp"
	"within.website/x/internal"
)

var (
	bind    = flag.String("bind", ":2934", "HTTP bind address")
	dataDir = flag.String("data-dir", "./var", "location to store data persistently")
)

func main() {
	internal.HandleStartup()

	slog.Info("starting up", "bind", *bind, "data-dir", *dataDir)

	db, err := connectDB()
	if err != nil {
		log.Fatalf("can't connect to DB: %v", err)
	}
	_ = db

	s := &Server{db: db}

	compress1KB := connect.WithCompressMinBytes(1024)

	mux := http.NewServeMux()
	mux.Handle(patchouliconnect.NewSyndicateHandler(s, compress1KB))

	slog.Info("listening", "url", "http://localhost"+*bind)
	log.Fatal(http.ListenAndServe(*bind, mux))
}

func connectDB() (*gorm.DB, error) {
	db, err := gorm.Open(gormlite.Open(filepath.Join(*dataDir, "data.db")), &gorm.Config{
		Logger: slogGorm.New(
			slogGorm.WithErrorField("err"),
			slogGorm.WithRecordNotFoundError(),
		),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.AutoMigrate(
		&Video{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	db.Use(gormPrometheus.New(gormPrometheus.Config{
		DBName: "mi",
	}))

	return db, nil
}

type Server struct {
	db *gorm.DB
}

func (s *Server) Info(
	ctx context.Context,
	req *connect.Request[patchouli.TwitchInfoReq],
) (
	*connect.Response[patchouli.TwitchInfoResp],
	error,
) {
	metadata, err := ytdlp.Metadata(ctx, req.Msg.GetUrl())
	if err != nil {
		slog.Error("can't fetch metadata for video", "url", req.Msg.GetUrl(), "err", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	result := &patchouli.TwitchInfoResp{
		Id:           metadata.ID,
		Title:        metadata.Title,
		ThumbnailUrl: metadata.Thumbnail,
		Duration:     metadata.DurationString,
		UploadDate:   timestamppb.New(metadata.UploadDate.Time),
		Url:          req.Msg.GetUrl(),
	}

	return connect.NewResponse(result), nil
}

func (s *Server) Download(
	ctx context.Context,
	req *connect.Request[patchouli.TwitchDownloadReq],
) (
	*connect.Response[patchouli.TwitchDownloadResp],
	error,
) {
	dir := filepath.Join(*dataDir, "video")

	if err := ytdlp.Download(ctx, req.Msg.GetUrl(), dir); err != nil {
		return nil, err
	}

	result := &patchouli.TwitchDownloadResp{
		Url:      req.Msg.Url,
		Location: dir,
	}

	return connect.NewResponse(result), nil
}

type Video struct {
	gorm.Model
	TwitchID  string `gorm:"uniqueIndex"`
	TwitchURL string `gorm:"uniqueIndex"`
	Title     string
	State     string
	BlogPath  sql.NullString
}
