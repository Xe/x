// Command uploud is an automated Backblaze B2 uploader for my infra.
package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"within.website/x/internal"
	"within.website/x/internal/avif"
)

var (
	b2Bucket = flag.String("tigris-bucket", "xedn", "Tigris bucket to dump things to")

	avifQuality      = flag.Int("avif-quality", 24, "AVIF quality (higher is worse quality)")
	avifEncoderSpeed = flag.Int("avif-encoder-speed", 0, "AVIF encoder speed (higher is faster)")

	jpegQuality = flag.Int("jpeg-quality", 85, "JPEG quality (lower means lower file size)")

	webpQuality = flag.Int("webp-quality", 50, "WEBP quality (higher is worse quality)")

	noEncode = flag.Bool("no-encode", false, "if set, just upload the file directly without encoding")
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

	s3c, err := internal.TigrisClient(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for _, finfo := range files {
		log.Printf("uploading %s", finfo.Name())
		fin, err := os.Open(filepath.Join(td, finfo.Name()))
		if err != nil {
			log.Fatal(err)
		}
		defer fin.Close()

		_, err = s3c.PutObject(ctx, &s3.PutObjectInput{
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
	".wasm": "application/wasm",
	".css":  "text/css",
}
