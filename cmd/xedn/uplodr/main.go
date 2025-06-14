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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/avif"
	_ "github.com/gen2brain/heic"
	"github.com/gen2brain/jpegxl"
	"github.com/gen2brain/webp"
	"google.golang.org/grpc"
	pb "within.website/x/gen/within/website/x/xedn/uplodr/v1"
	"within.website/x/internal"
	"within.website/x/tigris"
)

var (
	grpcAddr = flag.String("grpc-addr", ":8080", "address to listen on for GRPC")

	b2Bucket    = flag.String("b2-bucket", "christine-static", "Backblaze B2 bucket to dump things to")
	b2KeyID     = flag.String("b2-key-id", "", "Backblaze B2 application key ID")
	b2KeySecret = flag.String("b2-application-key", "", "Backblaze B2 application secret")

	msgSize = flag.Int("msg-size", 100*1024*1024, "how big the message should be")

	tigrisBucket = flag.String("bucket-name", "xedn", "Tigris bucket to dump things to")

	avifEncoderSpeed = flag.Int("avif-encoder-speed", 0, "AVIF encoder speed (higher is faster)")
	jxlEffort        = flag.Int("jxl-effort", 7, "JPEG XL encoding effort in the range [1,10]. Sets encoder effort/speed level without affecting decoding speed. Default is 7.")
	imageQuality     = flag.Int("image-quality", 85, "image quality (lower means lower file size)")
	webpMethod       = flag.Int("webp-method", 4, "WebP encoding method (0-6, 0 is fastest-worst, 6 is slowest-best)")
)

func main() {
	internal.HandleStartup()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	os.MkdirAll("/tmp", 0777)

	s, err := New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		<-ctx.Done()
		slog.Error("timeout")
		os.Exit(0)
	}()

	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Fatal(err)
	}

	gs := grpc.NewServer(
		grpc.MaxRecvMsgSize(*msgSize),
		grpc.MaxSendMsgSize(*msgSize),
	)

	pb.RegisterImageServer(gs, s)

	log.Fatal(gs.Serve(lis))
}

type Server struct {
	tc  *s3.Client
	b2c *s3.Client

	pb.UnimplementedImageServer
}

func New(ctx context.Context) (*Server, error) {
	tc, err := tigris.Client(ctx)
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
	slog.Info("ping", "msg", msg.Nonce)
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
	os.MkdirAll(dir, 0777)

	name := baseName + "-smol.png"
	fnames = append(fnames, name)
	if err := resizeSmol(img, filepath.Join(dir, name)); err != nil {
		return nil, fmt.Errorf("failed to make smol png: %w", err)
	}
	slog.Debug("converted", "name", name)

	name = baseName + ".webp"
	fnames = append(fnames, name)
	if err := doWEBP(img, filepath.Join(dir, name)); err != nil {
		return nil, fmt.Errorf("failed to make webp: %w", err)
	}
	slog.Debug("converted", "name", name)

	name = baseName + ".avif"
	fnames = append(fnames, name)
	if err := doAVIF(img, filepath.Join(dir, name)); err != nil {
		return nil, fmt.Errorf("failed to make avif: %w", err)
	}
	slog.Debug("converted", "name", name)

	name = baseName + ".jpg"
	fnames = append(fnames, name)
	if err := doJPEG(img, filepath.Join(dir, name)); err != nil {
		return nil, fmt.Errorf("failed to make jpeg: %w", err)
	}
	slog.Debug("converted", "name", name)

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

		key := fmt.Sprintf("%s/%s", ur.Folder, fname)

		if _, err := s.tc.PutObject(ctx, &s3.PutObjectInput{
			Body:        fin,
			Bucket:      tigrisBucket,
			Key:         aws.String(key),
			ContentType: aws.String(mimeTypes[filepath.Ext(fname)]),
		}); err != nil {
			slog.Error("can't upload", "to", "tigris", "err", err)
			errs = append(errs, fmt.Errorf("while uploading %s to tigris: %w", path, err))
			continue
		}
		slog.Debug("uploaded", "to", "tigris", "key", key)

		fin.Seek(0, 0)

		if _, err := s.b2c.PutObject(ctx, &s3.PutObjectInput{
			Body:        fin,
			Bucket:      b2Bucket,
			Key:         aws.String(key),
			ContentType: aws.String(mimeTypes[filepath.Ext(fname)]),
		}); err != nil {
			slog.Error("can't upload", "to", "b2", "err", err)
			errs = append(errs, fmt.Errorf("while uploading %s to b2: %w", path, err))
			continue
		}
		slog.Debug("uploaded", "to", "b2", "key", key)

		result = append(result, &pb.Variant{
			Url:      fmt.Sprintf("https://cdn.xeiaso.net/file/christine-static/%s/%s", ur.GetFolder(), fname),
			MimeType: mimeTypes[filepath.Ext(fname)],
		})
	}

	if len(errs) != 0 {
		return nil, errors.Join(errs...)
	}

	slog.Info("uploaded", "input", ur.FileName, "result", result)

	return &pb.UploadResp{
		Variants: result,
	}, nil
}

func doAVIF(src image.Image, dstPath string) error {
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("can't create destination file %s: %v", dstPath, err)
	}
	defer dst.Close()

	if err := avif.Encode(dst, src, avif.Options{
		Quality:      *imageQuality,
		QualityAlpha: *imageQuality,
		Speed:        *avifEncoderSpeed,
	}); err != nil {
		return err
	}

	log.Printf("Encoded AVIF at %s", dstPath)

	return nil
}

func doJXL(src image.Image, dstPath string) error {
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Fatalf("Can't create destination file: %v", err)
	}
	defer dst.Close()

	err = jpegxl.Encode(dst, src, jpegxl.Options{
		Quality: *imageQuality,
		Effort:  *jxlEffort,
	})

	if err != nil {
		return err
	}

	log.Printf("Encoded JPEG XL at %s", dstPath)

	return nil
}

func doWEBP(src image.Image, dstPath string) error {
	fout, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("can't create destination file %s: %v", dstPath, err)
	}
	defer fout.Close()

	err = webp.Encode(fout, src, webp.Options{
		Quality: *imageQuality,
		Method:  *webpMethod,
	})
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
		return fmt.Errorf("can't create destination file %s: %v", dstPath, err)
	}
	defer fout.Close()

	if err := jpeg.Encode(fout, src, &jpeg.Options{
		Quality: *imageQuality,
	}); err != nil {
		return err
	}

	log.Printf("Encoded JPEG at %s", dstPath)

	return nil
}

func resizeSmol(src image.Image, dstPath string) error {
	fout, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("can't create destination file %s: %v", dstPath, err)
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

func mkB2Client() *s3.Client {
	s3Config := aws.Config{
		Credentials:  credentials.NewStaticCredentialsProvider(*b2KeyID, *b2KeySecret, ""),
		BaseEndpoint: aws.String("https://s3.us-west-001.backblazeb2.com"),
		Region:       "us-west-001",
	}
	s3Client := s3.NewFromConfig(s3Config, (func(o *s3.Options) {
		o.UsePathStyle = true
		o.Region = "us-west-001"
	}))
	return s3Client
}
