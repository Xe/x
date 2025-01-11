// Command uploud is an automated Backblaze B2 uploader for my infra.
package main

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/disintegration/imaging"
	"github.com/gen2brain/avif"
	_ "github.com/gen2brain/heic"
	"github.com/gen2brain/jpegxl"
	"github.com/gen2brain/webp"
	"golang.org/x/sync/errgroup"
	"within.website/x/internal"
)

var (
	avifEncoderSpeed = flag.Int("avif-encoder-speed", 0, "AVIF encoder speed (higher is faster)")
	jxlEffort        = flag.Int("jxl-effort", 7, "JPEG XL encoding effort in the range [1,10]. Sets encoder effort/speed level without affecting decoding speed. Default is 7.")
	imageQuality     = flag.Int("image-quality", 85, "image quality (lower means lower file size)")
	tigrisBucket     = flag.String("tigris-bucket", "xedn", "Tigris bucket to dump things to")
	webpMethod       = flag.Int("webp-method", 4, "WebP encoding method (0-6, 0 is fastest-worst, 6 is slowest-best)")

	noEncode = flag.Bool("no-encode", false, "if set, just upload the file directly without encoding")
)

func main() {
	internal.HandleStartup()

	if flag.NArg() != 2 {
		log.Fatalf("usage: %s <filename/folder> <b2 path>", os.Args[0])
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	td, err := os.MkdirTemp("", "uploud")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(td)

	st, err := os.Stat(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}

	if st.IsDir() {
		files, err := os.ReadDir(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}

		for _, finfo := range files {
			if !*noEncode {
				if err := processImage(filepath.Join(flag.Arg(0), finfo.Name()), td); err != nil {
					log.Fatal(err)
				}
			} else {
				if _, err := copyFile(filepath.Join(flag.Arg(0), finfo.Name()), filepath.Join(td, finfo.Name())); err != nil {
					log.Fatal(err)
				}
			}
		}
	} else {
		if !*noEncode {
			if err := processImage(flag.Arg(0), td); err != nil {
				log.Fatal(err)
			}
		} else {
			if _, err := copyFile(flag.Arg(0), filepath.Join(td, filepath.Base(flag.Arg(0)))); err != nil {
				log.Fatal(err)
			}
		}
	}

	files, err := os.ReadDir(td)
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	s3c := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	for _, finfo := range files {
		log.Printf("uploading %s", finfo.Name())
		fin, err := os.Open(filepath.Join(td, finfo.Name()))
		if err != nil {
			log.Fatal(err)
		}
		defer fin.Close()

		st, err := fin.Stat()
		if err != nil {
			log.Fatal(err)
		}

		shaSum, err := hashFileSha256(fin)
		if err != nil {
			log.Fatal(err)
		}

		md5Sum, err := hashFileMD5(fin)
		if err != nil {
			log.Fatal(err)
		}

		_, err = s3c.PutObject(ctx,
			&s3.PutObjectInput{
				Body:           fin,
				Bucket:         tigrisBucket,
				Key:            aws.String(flag.Arg(1) + "/" + finfo.Name()),
				ContentType:    aws.String(mimeTypes[filepath.Ext(finfo.Name())]),
				ContentLength:  aws.Int64(st.Size()),
				ChecksumSHA256: aws.String(shaSum),
				ContentMD5:     aws.String(md5Sum),
				CacheControl:   aws.String("max-age=2592000,public"),
			},
			//tigris.WithCreateObjectIfNotExists(),
		)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func doAVIF(src image.Image, dstPath string) error {
	dst, err := os.Create(dstPath)
	if err != nil {
		log.Fatalf("Can't create destination file: %v", err)
	}
	defer dst.Close()

	err = avif.Encode(dst, src, avif.Options{
		Quality:      *imageQuality,
		QualityAlpha: *imageQuality,
		Speed:        *avifEncoderSpeed,
	})

	if err != nil {
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
		return err
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
		return err
	}
	defer fout.Close()

	if err := jpeg.Encode(fout, src, &jpeg.Options{Quality: *imageQuality}); err != nil {
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

	dstImg := src
	if src.Bounds().Dx() > 800 {
		dstImg = imaging.Resize(src, 800, 0, imaging.Lanczos)
	}

	enc := png.Encoder{
		CompressionLevel: png.BestCompression,
	}

	if err := enc.Encode(fout, dstImg); err != nil {
		return err
	}

	log.Printf("Encoded smol PNG at %s", dstPath)

	return nil
}

func processImage(fname, tempDir string) error {
	fnameBase := fileNameWithoutExt(fname)
	fin, err := os.Open(fname)
	if err != nil {
		return err
	}

	src, _, err := image.Decode(fin)
	if err != nil {
		return fmt.Errorf("decoding image %s: %w", fname, err)
	}

	eg, _ := errgroup.WithContext(context.Background())

	eg.Go(func() error {
		if err := doAVIF(src, filepath.Join(tempDir, fnameBase+".avif")); err != nil {
			return fmt.Errorf("avif: %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		if err := doJXL(src, filepath.Join(tempDir, fnameBase+".jxl")); err != nil {
			return fmt.Errorf("jxl: %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		if err := doWEBP(src, filepath.Join(tempDir, fnameBase+".webp")); err != nil {
			return fmt.Errorf("webp: %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		if err := doJPEG(src, filepath.Join(tempDir, fnameBase+".jpg")); err != nil {
			return fmt.Errorf("jpeg: %w", err)
		}

		return nil
	})

	eg.Go(func() error {
		if err := resizeSmol(src, filepath.Join(tempDir, fnameBase+"-smol.png")); err != nil {
			return fmt.Errorf("smol: %w", err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

var mimeTypes = map[string]string{
	".avif":  "image/avif",
	".webp":  "image/webp",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".heic":  "image/heic",
	".jxl":   "image/jxl",
	".png":   "image/png",
	".svg":   "image/svg+xml",
	".wasm":  "application/wasm",
	".css":   "text/css",
	".ts":    "video/mp2t",
	".js":    "application/javascript",
	".html":  "text/html",
	".json":  "application/json",
	".txt":   "text/plain",
	".md":    "text/markdown",
	".xml":   "application/xml",
	".zip":   "application/zip",
	".gz":    "application/gzip",
	".tar":   "application/x-tar",
	".pdf":   "application/pdf",
	".mp4":   "video/mp4",
	".webm":  "video/webm",
	".ogg":   "audio/ogg",
	".mp3":   "audio/mpeg",
	".wav":   "audio/wav",
	".flac":  "audio/flac",
	".aac":   "audio/aac",
	".m4a":   "audio/mp4",
	".opus":  "audio/opus",
	".ico":   "image/x-icon",
	".otf":   "font/otf",
	".ttf":   "font/ttf",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".eot":   "application/vnd.ms-fontobject",
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

// hashFileMD5 hashes a file with MD5 and returns the hash as a base64 encoded string.
func hashFileMD5(fin *os.File) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, fin); err != nil {
		return "", err
	}

	// rewind the file
	if _, err := fin.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}
