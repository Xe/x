package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"buf.build/go/protovalidate"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	pb "within.website/x/gen/within/website/x/futuresight/v1"
	"within.website/x/internal"
	"within.website/x/internal/xesite"
	"within.website/x/web/useragent"
)

var (
	apiBind = flag.String("api-bind", ":8080", "address to bind API to")
	bind    = flag.String("bind", ":8081", "address to bind zipserver to")

	awsAccessKeyID = flag.String("aws-access-key-id", "", "AWS access key ID")
	awsSecretKey   = flag.String("aws-secret-access-key", "", "AWS secret access key")
	awsEndpointS3  = flag.String("aws-endpoint-url-s3", "http://localhost:9000", "AWS S3 endpoint")
	awsRegion      = flag.String("aws-region", "auto", "AWS region")
	bucketName     = flag.String("bucket-name", "xesite-preview-versions", "bucket to fetch previews from")
	dataDir        = flag.String("data-dir", "./var", "directory to store data in (not permanent)")
	natsURL        = flag.String("nats-url", "nats://localhost:4222", "nats url")
	usePathStyle   = flag.Bool("use-path-style", false, "use path style for S3")
	valkeyHost     = flag.String("valkey-host", "localhost:6379", "host:port for valkey")
	valkeyPassword = flag.String("valkey-password", "", "password for valkey")
)

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	creds := credentials.NewStaticCredentialsProvider(*awsAccessKeyID, *awsSecretKey, "")

	s3c := s3.New(s3.Options{
		AppID:            useragent.GenUserAgent("future-sight-push", "https://xeiaso.net"),
		BaseEndpoint:     awsEndpointS3,
		ClientLogMode:    aws.LogRetries | aws.LogRequest | aws.LogResponse,
		Credentials:      creds,
		EndpointResolver: s3.EndpointResolverFromURL(*awsEndpointS3),
		//Logger:           logging.NewStandardLogger(os.Stderr),
		UsePathStyle: *usePathStyle,
		Region:       *awsRegion,
	})

	slog.Debug("details",
		"awsAccessKeyID", *awsAccessKeyID,
		"awsSecretKey", *awsSecretKey,
		"awsEndpointS3", *awsEndpointS3,
		"awsRegion", *awsRegion,
		"bucketName", *bucketName,
		"natsURL", *natsURL,
		"usePathStyle", *usePathStyle,
		"valkeyHost", *valkeyHost,
		"valkeyPassword", *valkeyPassword,
	)

	vk := redis.NewClient(&redis.Options{
		Addr:     *valkeyHost,
		Password: *valkeyPassword,
		DB:       0,
	})
	defer vk.Close()

	if _, err := vk.Ping(context.Background()).Result(); err != nil {
		log.Fatal(err)
	}

	nc, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	zs, err := xesite.NewZipServer(filepath.Join(*dataDir, "current.zip"), *dataDir)
	if err != nil {
		log.Fatal(err)
	}

	s := &Server{
		s3c: s3c,
		vk:  vk,
		nc:  nc,
		zs:  zs,
		dir: *dataDir,
	}

	currentVersion, err := vk.Get(ctx, "future-sight:current").Result()
	if err != nil && err != redis.Nil {
		log.Fatal(err)
	}

	if currentVersion != "" {
		nv := pb.NewVersion{
			Slug: currentVersion,
		}

		if err := s.fetchVersion(ctx, &nv); err != nil {
			slog.Error("can't fetch current version", "err", err)
		}
	}

	if _, err := nc.Subscribe("future-sight-push", s.HandleFutureSightPushMsg); err != nil {
		log.Fatal(err)
	}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/upload", s.UploadVersion)
	apiMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	g, _ := errgroup.WithContext(context.Background())

	g.Go(func() error {
		slog.Info("listening", "for", "api", "addr", *apiBind)
		return http.ListenAndServe(*apiBind, apiMux)
	})

	g.Go(func() error {
		slog.Info("listening", "for", "zipserver", "addr", *bind)
		return http.ListenAndServe(*bind, zs)
	})

	if err := g.Wait(); err != nil {
		slog.Error("error doing work", "err", err)
		os.Exit(1)
	}
}

type Server struct {
	s3c *s3.Client
	vk  *redis.Client
	nc  *nats.Conn
	zs  *xesite.ZipServer
	dir string
}

func (s *Server) UploadVersion(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	slog.Info("uploading version")

	if err := r.ParseMultipartForm(10 << 24); err != nil {
		slog.Error("failed to parse form", "err", err)
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	f, header, err := r.FormFile("file")
	if err != nil {
		slog.Error("failed to get file", "err", err)
		http.Error(w, "failed to get file", http.StatusBadRequest)
		return
	}
	defer f.Close()

	slog.Info("got file", "filename", header.Filename)

	fout, err := os.CreateTemp(s.dir, "future-sight-upload-*")
	if err != nil {
		slog.Error("failed to create temp file", "err", err)
		http.Error(w, "failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer fout.Close()
	defer os.Remove(fout.Name())

	if _, err := io.Copy(fout, f); err != nil {
		slog.Error("failed to copy file", "err", err)
		http.Error(w, "failed to copy file", http.StatusInternalServerError)
		return
	}

	fout.Seek(0, 0)

	hash, err := hashFileSha256(fout)
	if err != nil {
		slog.Error("failed to hash file", "err", err)
		http.Error(w, "failed to hash file", http.StatusInternalServerError)
		return
	}

	st, err := fout.Stat()
	if err != nil {
		slog.Error("failed to stat file", "err", err)
		http.Error(w, "failed to stat file", http.StatusInternalServerError)
		return
	}

	slog.Info("hashed file", "hash", hash)

	if _, err := s.s3c.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        bucketName,
		Key:           aws.String(hash),
		Body:          fout,
		ContentType:   aws.String("application/zip"),
		ContentLength: aws.Int64(st.Size()),
		Metadata: map[string]string{
			"host_os": runtime.GOOS,
		},
	}); err != nil {
		slog.Error("failed to push file", "bucketName", *bucketName, "hash", hash, "err", err)
		http.Error(w, "failed to push file", http.StatusInternalServerError)
		return
	}

	nv := &pb.NewVersion{
		Slug: hash,
	}

	if err := s.PushVersion(ctx, nv); err != nil {
		slog.Error("failed to push version", "slug", nv.Slug, "err", err)
		http.Error(w, "failed to push version", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) PushVersion(ctx context.Context, nv *pb.NewVersion) error {
	slog.Info("got new version", "version", nv)

	msg, err := proto.Marshal(nv)
	if err != nil {
		slog.Error("failed to marshal message", "slug", nv.Slug, "err", err)
		return err
	}

	if err := s.nc.Publish("future-sight-push", msg); err != nil {
		slog.Error("failed to publish message", "slug", nv.Slug, "err", err)
		return err
	}

	if _, err := s.vk.Set(ctx, "future-sight:current", nv.Slug, 0).Result(); err != nil {
		slog.Error("failed to set current version", "slug", nv.Slug, "err", err)
		return err
	}

	return nil
}

func (s *Server) HandleFutureSightPushMsg(msg *nats.Msg) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nv := new(pb.NewVersion)
	if err := proto.Unmarshal(msg.Data, nv); err != nil {
		slog.Error("failed to unmarshal message", "err", err)
		return
	}

	if err := protovalidate.Validate(nv); err != nil {
		slog.Error("failed to validate message", "err", err)
		return
	}

	if err := s.fetchVersion(ctx, nv); err != nil {
		slog.Error("failed to handle message", "slug", nv.Slug, "err", err)
		return
	}

	slog.Info("handled message", "slug", nv.Slug)
}

func (s *Server) fetchVersion(ctx context.Context, nv *pb.NewVersion) error {
	os.Remove(filepath.Join(s.dir, "current.zip"))

	fout, err := os.Create(filepath.Join(s.dir, "current.zip"))
	if err != nil {
		return err
	}
	defer fout.Close()

	obj, err := s.s3c.GetObject(ctx, &s3.GetObjectInput{
		Bucket: bucketName,
		Key:    aws.String(nv.Slug),
	})
	if err != nil {
		os.Remove(filepath.Join(s.dir, "current.zip"))
		slog.Error("failed to get object", "slug", nv.Slug, "err", err)
		return err
	}
	defer obj.Body.Close()

	if _, err := io.Copy(fout, obj.Body); err != nil {
		os.Remove(filepath.Join(s.dir, "current.zip"))
		slog.Error("failed to copy object", "slug", nv.Slug, "err", err)
		return err
	}

	if err := s.zs.Update(filepath.Join(s.dir, "current.zip")); err != nil {
		slog.Error("failed to update zipserver", "slug", nv.Slug, "err", err)
		return err
	}

	return nil
}

// hashFileSha256 hashes a file with Sha256 and returns the hash as a base64 encoded string.
func hashFileSha256(fin *os.File) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, fin); err != nil {
		return "", err
	}

	// rewind the file
	if _, err := fin.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
