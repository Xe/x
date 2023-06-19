package main

import (
	"bytes"
	"context"
	"crypto/sha512"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/bwmarrin/discordgo"
	_ "modernc.org/sqlite"
	"tailscale.com/hostinfo"
	"within.website/ln"
	"within.website/ln/opname"
	"within.website/x/bundler"
	"within.website/x/internal"
	"within.website/x/web"
	"within.website/x/web/revolt"
)

var (
	dbFile                = flag.String("db-file", "marabot.db", "Path to the database file")
	discordToken          = flag.String("discord-token", "", "Discord bot token")
	revoltToken           = flag.String("revolt-token", "", "Revolt bot token")
	revoltAPIServer       = flag.String("revolt-api-server", "https://api.revolt.chat", "API server for Revolt")
	revoltWebsocketServer = flag.String("revolt-ws-server", "wss://ws.revolt.chat", "Websocket server for Revolt")
	revoltBotID           = flag.String("revolt-bot-id", "", "bot ID for revolt")
	tsAuthkey             = flag.String("ts-authkey", "", "Tailscale authkey")
	tsHostname            = flag.String("ts-hostname", "", "Tailscale hostname")

	adminDiscordUser = flag.String("admin-discord-user", "", "Discord user ID of the admin")
	adminRevoltUser  = flag.String("admin-revolt-user", "", "Revolt user ID of the admin")

	furryholeDiscord = flag.String("furryhole-discord", "192289762302754817", "Discord channel ID for furryhole")
	furryholeRevolt  = flag.String("furryhole-revolt", "01FEXZ1XPWMEJXMF836FP16HB8", "Revolt channel ID for furryhole")
	awsS3Bucket      = flag.String("aws-s3-bucket", "", "S3 bucket name")
	awsS3Region      = flag.String("aws-s3-region", "ca-central-1", "S3 bucket region")

	//go:embed schema.sql
	dbSchema string
)

func main() {
	internal.HandleStartup()

	hostinfo.SetApp("within.website/x/cmd/marabot")

	ctx, cancel := context.WithCancel(opname.With(context.Background(), "marabot"))
	defer cancel()

	ln.Log(ctx, ln.Action("starting up"))

	db, err := sql.Open("sqlite", *dbFile)
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("opening sqlite database"))
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, dbSchema); err != nil {
		ln.FatalErr(ctx, err, ln.Action("running database schema"))
	}

	ircmsgs := make(chan string, 10)

	// Init a new client.
	client, err := revolt.NewWithEndpoint(*revoltToken, *revoltAPIServer, *revoltWebsocketServer)
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("creating revolt client"))
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(*awsS3Region)},
	)

	uploader := s3manager.NewUploader(sess)

	mr := &MaraRevolt{
		cli:      client,
		db:       db,
		ircmsgs:  ircmsgs,
		uploader: uploader,
		s3:       s3.New(sess),
	}

	mr.attachmentUpload = bundler.New(mr.S3Upload)
	mr.attachmentUpload.BundleCountThreshold = 5
	mr.attachmentUpload.DelayThreshold = time.Minute

	mr.attachmentPreprocess = bundler.New(mr.PreprocessLinks)
	mr.attachmentPreprocess.BundleCountThreshold = 10
	mr.attachmentPreprocess.DelayThreshold = 30 * time.Second

	go mr.IRCBot(ctx)

	client.Connect(ctx, mr)

	dg, err := discordgo.New("Bot " + *discordToken)
	if err != nil {
		ln.FatalErr(ctx, err, ln.Action("creating discord client"))
	}

	dg.AddHandler(mr.DiscordMessageCreate)
	dg.AddHandler(mr.DiscordMessageDelete)
	dg.AddHandler(mr.DiscordMessageEdit)

	if err := dg.Open(); err != nil {
		ln.FatalErr(ctx, err, ln.Action("opening discord client"))
	}
	defer dg.Close()

	if err := mr.importDiscordData(ctx, db, dg); err != nil {
		ln.Error(ctx, err)
	}

	// Wait for close.
	sc := make(chan os.Signal, 1)

	signal.Notify(
		sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		os.Interrupt,
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-sc:
			ln.Log(ctx, ln.Info("shutting down"))
			cancel()
			mr.attachmentPreprocess.Flush()
			mr.attachmentUpload.Flush()
			time.Sleep(150 * time.Millisecond)
			return
		}
	}
}

type Attachment struct {
	ID          string  `json:"id"`
	URL         string  `json:"url"`
	Kind        string  `json:"kind"`
	ContentType string  `json:"content_type"`
	CreatedAt   string  `json:"created_at"`
	MessageID   *string `json:"message_id"`
	Data        []byte  `json:"-"`
}

func (mr *MaraRevolt) PreprocessLinks(data [][3]string) {
	ctx := opname.With(context.Background(), "marabot.link-preprocessor")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	mr.preprocessLinks(ctx, data)
}

func (mr *MaraRevolt) preprocessLinks(ctx context.Context, data [][3]string) {
	for _, linkkind := range data {
		kind := linkkind[1]
		link := linkkind[0]
		msgID := linkkind[2]

		var count int
		if err := mr.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM s3_uploads WHERE url = ?", link).Scan(&count); err != nil {
			ln.Error(ctx, err)
			continue
		}
		if count != 0 {
			continue
		}

		att, err := hashURL(link, kind)
		if err != nil {
			ln.Error(ctx, err, ln.F{"link": link, "kind": kind})
			continue
		}

		att.MessageID = aws.String(msgID)

		mr.attachmentUpload.Add(att, len(att.Data))
	}
}

func hashURL(itemURL, kind string) (*Attachment, error) {
	resp, err := http.Get(itemURL)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, web.NewError(http.StatusOK, resp)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	h := sha512.New()
	h.Write(data)
	hash := fmt.Sprintf("%x", h.Sum(nil))

	result := &Attachment{
		ID:          hash,
		URL:         itemURL,
		Kind:        kind,
		CreatedAt:   time.Now().Format(time.RFC3339),
		ContentType: resp.Header.Get("Content-Type"),
		Data:        data,
	}

	return result, nil
}

func (mr *MaraRevolt) S3Upload(att []*Attachment) {
	ctx := opname.With(context.Background(), "marabot.s3-uploader")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	mr.s3Upload(ctx, att)
}

func (mr *MaraRevolt) s3Upload(ctx context.Context, att []*Attachment) {
	for _, att := range att {
		key := filepath.Join(att.Kind, att.ID)

		f := ln.F{"kind": att.Kind, "id": att.ID, "url": att.URL, "content_type": att.ContentType}

		var count int
		if err := mr.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM s3_uploads WHERE id = ?", att.ID).Scan(&count); err != nil {
			ln.Error(ctx, err, f)
			continue
		}

		f["count"] = count

		if count != 0 {
			continue
		}

		if _, err := mr.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
			Bucket:      aws.String(*awsS3Bucket),
			Key:         aws.String(key),
			ContentType: aws.String(att.ContentType),
			Body:        bytes.NewBuffer(att.Data),
			Metadata: map[string]*string{
				"Original-URL": aws.String(att.URL),
				"Message-ID":   att.MessageID,
			},
		}); err != nil {
			ln.Error(ctx, err, ln.Action("trying to upload to S3"), f)
			continue
		}

		if _, err := mr.db.ExecContext(ctx, "INSERT INTO s3_uploads(id, url, kind, content_type, created_at, message_id) VALUES (?, ?, ?, ?, ?, ?)", att.ID, att.URL, att.Kind, att.ContentType, att.CreatedAt, att.MessageID); err != nil {
			ln.Error(ctx, err, ln.Action("saving upload information to DB"), f)
		}
	}

}
