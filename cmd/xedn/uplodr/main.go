package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"google.golang.org/grpc"
	"within.website/x/cmd/xedn/uplodr/pb"
	"within.website/x/internal"
	"within.website/x/internal/avif"
)

var (
	grpcAddr = flag.String("grpc-addr", ":8080", "address to listen on for GRPC")

	b2Bucket    = flag.String("b2-bucket", "christine-static", "Backblaze B2 bucket to dump things to")
	b2KeyID     = flag.String("b2-key-id", "", "Backblaze B2 application key ID")
	b2KeySecret = flag.String("b2-application-key", "", "Backblaze B2 application secret")

	tigrisBucket = flag.String("bucket-name", "xedn", "Tigris bucket to dump things to")

	avifQuality      = flag.Int("avif-quality", 8, "AVIF quality (higher is worse quality)")
	avifEncoderSpeed = flag.Int("avif-encoder-speed", 0, "AVIF encoder speed (higher is faster)")

	jpegQuality = flag.Int("jpeg-quality", 90, "JPEG quality (lower means lower file size)")

	webpQuality = flag.Int("webp-quality", 9, "WEBP quality (higher is worse quality)")
)

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	s, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		<-ctx.Done()
		panic("timeout")
	}()

	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Fatal(err)
	}

	gs := grpc.NewServer()

	pb.RegisterImageServer(gs, s)

	log.Fatal(gs.Serve(lis))
}

type Server struct {
	tc  *s3.Client
	b2c *s3.Client

	pb.UnimplementedImageServer
}

func New(ctx context.Context) (*Server, error) {
	tc, err := mkTigrisClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Tigris client: %w", err)
	}

	b2c := mkB2Client()

	return &Server{
		tc:  tc,
		b2c: b2c,
	}, nil
}

func (s *Server) Ping(ctx context.Context, msg *pb.Echo) (*pb.Echo, error) {
	return msg, nil
}

func (s *Server) Upload(ctx context.Context, ur *pb.UploadReq) (*pb.UploadResp, error) {
	img, format, err := image.Decode(bytes.NewBuffer(ur.Data))
	if err != nil {
		slog.Error("can't decode image", "err", err, "filename", ur.GetFileName())
		return nil, err
	}
	slog.Debug("got image", "format", format)

	baseName := fileNameWithoutExt(ur.GetFileName())

	fnames := []string{}

	dir, err := os.MkdirTemp("", "xedn-uplodr")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	name := baseName + "-smol.png"
	fnames = append(fnames, name)
	if err := resizeSmol(img, filepath.Join(dir, name)); err != nil {
		return nil, fmt.Errorf("failed to make smol png: %w", err)
	}

	name = baseName + ".webp"
	fnames = append(fnames, filepath.Join(dir, name))
	if err := doWEBP(img, name); err != nil {
		return nil, fmt.Errorf("failed to make webp: %w", err)
	}

	name = baseName + ".avif"
	fnames = append(fnames, filepath.Join(dir, name))
	if err := doAVIF(img, name); err != nil {
		return nil, fmt.Errorf("failed to make avif: %w", err)
	}

	name = baseName + ".jpg"
	fnames = append(fnames, filepath.Join(dir, name))
	if err := doJPEG(img, name); err != nil {
		return nil, fmt.Errorf("failed to make jpeg: %w", err)
	}

	var result []*pb.Variant

	errs := []error{}
	for _, fname := range fnames {
		path := filepath.Join(dir, fname)
		slog.Info("uploading", "path", path)

		fin, err := os.Open(path)
		if err != nil {
			slog.Error("can't open file", "path", path, "err", err)
			errs = append(errs, fmt.Errorf("while uploading %s: %w", path, err))
			continue
		}
		defer fin.Close()

		if _, err := s.b2c.PutObject(ctx, &s3.PutObjectInput{
			Body:        fin,
			Bucket:      b2Bucket,
			Key:         aws.String(fmt.Sprintf("%s/%s", ur.Folder, fname)),
			ContentType: aws.String(mimeTypes[filepath.Ext(fname)]),
		}); err != nil {
			slog.Error("can't upload", "to", "b2", "err", err)
			errs = append(errs, fmt.Errorf("while uploading %s to b2: %w", path, err))
			continue
		}

		fin.Seek(0, 0)

		if _, err := s.tc.PutObject(ctx, &s3.PutObjectInput{
			Body:        fin,
			Bucket:      b2Bucket,
			Key:         aws.String(fmt.Sprintf("%s/%s", ur.Folder, fname)),
			ContentType: aws.String(mimeTypes[filepath.Ext(fname)]),
		}); err != nil {
			slog.Error("can't upload", "to", "b2", "err", err)
			errs = append(errs, fmt.Errorf("while uploading %s to b2: %w", path, err))
			continue
		}

		result = append(result, &pb.Variant{
			Url:      fmt.Sprintf("https://cdn.xeiaso.net/file/christine-static/%s/%s", ur.GetFolder(), fname),
			MimeType: mimeTypes[filepath.Ext(fname)],
		})
	}

	if len(errs) != 0 {
		return nil, errors.Join(errs...)
	}

	return &pb.UploadResp{
		Variants: result,
	}, nil
}

func doAVIF(src image.Image, dstPath string) error {
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Fatalf("Can't create destination file: %v", err)
	}
	defer dst.Close()

	err = avif.Encode(dst, src, &avif.Options{
		Threads: runtime.GOMAXPROCS(0),
		Speed:   *avifEncoderSpeed,
		Quality: *avifQuality,
	})
	if err != nil {
		return err
	}

	log.Printf("Encoded AVIF at %s", dstPath)

	return nil
}

func doWEBP(src image.Image, dstPath string) error {
	fout, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fout.Close()

	err = webp.Encode(fout, src, &webp.Options{Quality: float32(*webpQuality)})
	if err != nil {
		return err
	}

	log.Printf("Encoded WEBP at %s", dstPath)

	return nil
}

func fileNameWithoutExt(fileName string) string {
	return filepath.Base(fileName[:len(fileName)-len(filepath.Ext(fileName))])
}

func doJPEG(src image.Image, dstPath string) error {
	fout, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fout.Close()

	if err := jpeg.Encode(fout, src, &jpeg.Options{Quality: *jpegQuality}); err != nil {
		return err
	}

	log.Printf("Encoded JPEG at %s", dstPath)

	return nil
}

func resizeSmol(src image.Image, dstPath string) error {
	fout, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer fout.Close()

	dstImg := imaging.Resize(src, 800, 0, imaging.Lanczos)

	enc := png.Encoder{
		CompressionLevel: png.BestCompression,
	}

	if err := enc.Encode(fout, dstImg); err != nil {
		return err
	}

	log.Printf("Encoded smol PNG at %s", dstPath)

	return nil
}

var mimeTypes = map[string]string{
	".avif": "image/avif",
	".webp": "image/webp",
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".wasm": "application/wasm",
	".css":  "text/css",
}

func mkTigrisClient(ctx context.Context) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load Tigris config: %w", err)
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("https://fly.storage.tigris.dev")
	}), nil
}

func mkB2Client() *s3.Client {
	s3Config := aws.Config{
		Credentials:  credentials.NewStaticCredentialsProvider(*b2KeyID, *b2KeySecret, ""),
		BaseEndpoint: aws.String("https://s3.us-west-001.backblazeb2.com"),
		Region:       "us-west-001",
	}
	s3Client := s3.NewFromConfig(s3Config, (func(o *s3.Options) {
		o.UsePathStyle = true
	}))
	return s3Client
}
