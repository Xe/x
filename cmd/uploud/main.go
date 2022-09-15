// Command uploud is an automated Backblaze B2 uploader for my infra.
package main

import (
	"flag"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"tulpa.dev/cadey/avif"
)

var (
	b2Bucket = flag.String("b2-bucket", "christine-static", "Backblaze B2 bucket to dump things to")

	avifQuality      = flag.Int("avif-quality", 48, "AVIF quality (higher is worse quality)")
	avifEncoderSpeed = flag.Int("avif-encoder-speed", 0, "AVIF encoder speed (higher is faster)")

	jpegQuality = flag.Int("jpeg-quality", 85, "JPEG quality (lower means lower file size)")

	webpQuality = flag.Int("webp-quality", 75, "WEBP quality (higher is worse quality)")
)

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

func processImage(fname, tempDir string) error {
	fnameBase := fileNameWithoutExt(fname)
	fin, err := os.Open(fname)
	if err != nil {
		return err
	}

	src, _, err := image.Decode(fin)
	if err != nil {
		return err
	}

	if err := doAVIF(src, filepath.Join(tempDir, fnameBase+".avif")); err != nil {
		return err
	}

	if err := doWEBP(src, filepath.Join(tempDir, fnameBase+".webp")); err != nil {
		return err
	}

	if err := doJPEG(src, filepath.Join(tempDir, fnameBase+".jpg")); err != nil {
		return err
	}

	if err := resizeSmol(src, filepath.Join(tempDir, fnameBase+"-smol.png")); err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		log.Fatalf("usage: %s <filename/folder> <b2 path>", os.Args[0])
	}

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
			if err := processImage(filepath.Join(flag.Arg(0), finfo.Name()), td); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		if err := processImage(flag.Arg(0), td); err != nil {
			log.Fatal(err)
		}
	}

	files, err := os.ReadDir(td)
	if err != nil {
		log.Fatal(err)
	}

	s3c := mkS3Client()

	for _, finfo := range files {
		log.Printf("uploading %s", finfo.Name())
		fin, err := os.Open(filepath.Join(td, finfo.Name()))
		if err != nil {
			log.Fatal(err)
		}
		defer fin.Close()

		_, err = s3c.PutObject(&s3.PutObjectInput{
			Body:        fin,
			Bucket:      b2Bucket,
			Key:         aws.String(flag.Arg(1) + "/" + finfo.Name()),
			ContentType: aws.String(mimeTypes[filepath.Ext(finfo.Name())]),
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}

var mimeTypes = map[string]string{
	".avif": "image/avif",
	".webp": "image/webp",
	".jpg":  "image/jpeg",
	".png":  "image/png",
}

func mkS3Client() *s3.S3 {
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("B2_KEY_ID"), os.Getenv("B2_APPLICATION_KEY"), ""),
		Endpoint:         aws.String("https://s3.us-west-001.backblazeb2.com"),
		Region:           aws.String("us-west-001"),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession := session.New(s3Config)

	s3Client := s3.New(newSession)
	return s3Client
}
