package main

import (
	"bytes"
	"context"
	"expvar"
	"fmt"
	"image"
	"image/jpeg"
	"log/slog"
	"math/rand"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.etcd.io/bbolt"
	"golang.org/x/sync/singleflight"
	"within.website/x/web/stablediffusion"
)

var (
	stableDiffusionHits     = expvar.NewInt("xedn_avatar_hits")
	stableDiffusionCreation = expvar.NewInt("xedn_avatar_creation")
)

type StableDiffusion struct {
	client *stablediffusion.Client
	db     *bbolt.DB
	group  *singleflight.Group
}

// RenderImage renders a stable diffusion image based on the hash given.
//
// It assumes that the image does not exist, if it does, you will need
// to check elsewhere.
func (sd *StableDiffusion) RenderImage(ctx context.Context, w http.ResponseWriter, hash string) error {
	prompt, seed := hallucinatePrompt(hash)

	slog.Info("generating new image", "prompt", prompt)

	imgsVal, err, _ := sd.group.Do(hash, func() (interface{}, error) {
		imgs, err := sd.client.Generate(ctx, stablediffusion.SimpleImageRequest{
			Prompt:         "masterpiece, best quality, " + prompt,
			NegativePrompt: "person in distance, worst quality, low quality, medium quality, deleted, lowres, comic, bad anatomy, bad hands, text, error, missing fingers, extra digit, fewer digits, cropped, jpeg artifacts, signature, watermark, username, blurry",
			Seed:           seed,
			SamplerName:    "DPM++ 2M Karras",
			BatchSize:      1,
			NIter:          1,
			Steps:          20,
			CfgScale:       7,
			Width:          256,
			Height:         256,
			SNoise:         1,

			OverrideSettingsRestoreAfterwards: true,
		})
		if err != nil {
			return nil, err
		}

		stableDiffusionCreation.Add(1)

		img, _, err := image.Decode(bytes.NewBuffer(imgs.Images[0]))
		if err != nil {
			return nil, err
		}

		buf := &bytes.Buffer{}

		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: 75}); err != nil {
			return nil, err
		}

		imgs.Images[0] = buf.Bytes()

		return imgs, nil
	})
	if err != nil {
		return err
	}
	imgs := imgsVal.(*stablediffusion.ImageResponse)

	slog.Info("done generating image", "prompt", prompt)

	if err := sd.db.Update(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte("avatars"))

		if err := bkt.Put([]byte(hash), []byte(imgs.Images[0])); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	w.Header().Set("content-type", "image/jpeg")
	w.Header().Set("content-length", fmt.Sprint(len(imgs.Images[0])))
	w.Header().Set("expires", time.Now().Add(30*24*time.Hour).Format(http.TimeFormat))
	w.Header().Set("Cache-Control", "max-age:2630000") // one month
	w.WriteHeader(http.StatusOK)
	w.Write(imgs.Images[0])

	return nil
}

var isHexRegex = regexp.MustCompile(`[-fA-F0-9]+$`)

func (sd *StableDiffusion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hash := filepath.Base(r.URL.Path)

	if !isHexRegex.MatchString(hash) {
		http.Error(w, "the input must be a hexadecimal string", http.StatusBadRequest)
		return
	}

	if len(hash) != 32 {
		http.Error(w, "this must be 32 characters", http.StatusBadRequest)
		return
	}

	if err := sd.db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("avatars")); err != nil {
			return err
		}
		return nil
	}); err != nil {
		http.Error(w, "can't access database", http.StatusInternalServerError)
		return
	}

	found := false

	sd.db.View(func(tx *bbolt.Tx) error {
		bkt := tx.Bucket([]byte("avatars"))
		data := bkt.Get([]byte(hash))
		found = data != nil

		if found {
			w.Header().Set("content-type", "image/png")
			w.Header().Set("content-length", fmt.Sprint(len(data)))
			w.Header().Set("expires", time.Now().Add(30*24*time.Hour).Format(http.TimeFormat))
			w.Header().Set("Cache-Control", "max-age:2630000") // one month
			w.WriteHeader(http.StatusOK)
			w.Write(data)
		}

		stableDiffusionHits.Add(1)

		return nil
	})

	if !found {
		if err := sd.RenderImage(r.Context(), w, hash); err != nil {
			slog.Error("can't render image", "err", err)
			http.Error(w, "cannot render image, sorry", http.StatusInternalServerError)
			return
		}
	}
}

func hallucinatePrompt(hash string) (string, int) {
	var sb strings.Builder
	if hash[0] > '0' && hash[0] <= '5' {
		fmt.Fprint(&sb, "1girl, ")
	} else {
		fmt.Fprint(&sb, "1guy, ")
	}

	switch hash[1] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "blonde, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "brown hair, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "red hair, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "black hair, ")
	default:
	}

	if hash[2] > '0' && hash[2] <= '5' {
		fmt.Fprint(&sb, "coffee shop, ")
	} else {
		fmt.Fprint(&sb, "landscape, outdoors, ")
	}

	if hash[3] > '0' && hash[3] <= '5' {
		fmt.Fprint(&sb, "hoodie, ")
	} else {
		fmt.Fprint(&sb, "sweatsuit, ")
	}

	switch hash[4] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "<lora:cdi:1>, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "breath of the wild, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "genshin impact, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "arknights, ")
	default:
	}

	if hash[5] > '0' && hash[5] <= '5' {
		fmt.Fprint(&sb, "watercolor, ")
	} else {
		fmt.Fprint(&sb, "matte painting, ")
	}

	switch hash[6] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "highly detailed, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "ornate, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "thick lines, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "3d render, ")
	default:
	}

	switch hash[7] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "short hair, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "long hair, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "ponytail, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "pigtails, ")
	default:
	}

	switch hash[8] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "smile, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "frown, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "laughing, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "angry, ")
	default:
	}

	switch hash[9] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "sweater, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "tshirt, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "suitjacket, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "armor, ")
	default:
	}

	switch hash[10] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "blue eyes, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "red eyes, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "brown eyes, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "hazel eyes, ")
	default:
	}

	if hash[11] == '0' {
		fmt.Fprint(&sb, "heterochromia, ")
	}

	switch hash[12] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "morning, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "afternoon, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "evening, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "nighttime, ")
	default:
	}

	if hash[13] == '0' {
		fmt.Fprint(&sb, "<lora:genshin:1>, genshin, ")
	}

	switch hash[14] {
	case '0', '1', '2', '3':
		fmt.Fprint(&sb, "vtuber, ")
	case '4', '5', '6', '7':
		fmt.Fprint(&sb, "anime, ")
	case '8', '9', 'a', 'b':
		fmt.Fprint(&sb, "studio ghibli, ")
	case 'c', 'd', 'e', 'f':
		fmt.Fprint(&sb, "cloverworks, ")
	default:
	}

	seedPortion := hash[len(hash)-9 : len(hash)-1]
	seed, err := strconv.ParseInt(seedPortion, 16, 32)
	if err != nil {
		seed = int64(rand.Int())
	}

	fmt.Fprint(&sb, "pants")

	return sb.String(), int(seed)
}
