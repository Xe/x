package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/rivo/uniseg"
	"within.website/x/internal"
	"within.website/x/web/bskybot"
	"within.website/x/web/mastosan"
)

var (
	bskyAuthkey = flag.String("bsky-authkey", "", "Bluesky authkey")
	bskyHandle  = flag.String("bsky-handle", "", "Bluesky handle/email")
	bskyPDS     = flag.String("bsky-pds", "https://bsky.social", "Bluesky PDS")

	afterStr = flag.String("after", "2017-04-22T04:59:14Z", "only consider posts after this point")
)

func main() {
	internal.HandleStartup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agent := bskybot.NewAgent(ctx, *bskyPDS, *bskyHandle, *bskyAuthkey)
	if err := agent.Connect(ctx); err != nil {
		log.Fatal(err)
	}

	if flag.NArg() != 1 {
		fmt.Println("Usage: mastodon2atproto [flags] <data dir>")
	}

	dir := flag.Arg(0)

	fin, err := os.Open(filepath.Join(dir, "outbox.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer fin.Close()

	after, err := time.Parse(time.RFC3339, *afterStr)
	if err != nil {
		log.Fatalf("can't parse after time: %v", err)
	}

	var oc OrderedCollection
	if err := json.NewDecoder(fin).Decode(&oc); err != nil {
		log.Fatal(err)
	}

	slog.Info("found items", "count", oc.TotalItems)

	var out []Object

	for _, item := range oc.OrderedItems {
		slog.Debug("found item", "type", item.Type)

		if item.Type == "Announce" {
			continue
		}

		var obj Object
		if err := json.Unmarshal(item.Object, &obj); err != nil {
			log.Fatal(err)
		}

		if obj.InReplyTo != nil {
			continue
		}

		text, err := mastosan.Text(ctx, obj.Content)
		if err != nil {
			log.Fatal(err)
		}

		if strings.HasPrefix(text, "@") {
			continue
		}

		if uniseg.GraphemeClusterCount(text) > 300 {
			continue
		}

		if obj.Published.Before(after) {
			continue
		}

		skip := false
		for _, att := range obj.Attachment {
			if !strings.HasPrefix(att.URL, "image/") {
				skip = true
			}
		}

		if skip {
			continue
		}

		obj.Content = text

		slog.Debug("found post", "text", text, "published", obj.Published)

		out = append(out, obj)
	}

	slog.Info("found candidate posts for migration", "len", len(out))

	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for _, obj := range out {
		<-t.C

		slog.Info("migrating post", "post", obj)

		pb := bskybot.NewPostBuilder(obj.Content).
			AtTime(obj.Published)

		if len(obj.Attachment) != 0 {
			var blobs []lexutil.LexBlob
			var images []bskybot.Image
			for _, att := range obj.Attachment {
				att.URL = strings.TrimPrefix(att.URL, "/interlinked-mst3k")
				fname := filepath.Join(dir, att.URL)

				st, err := os.Stat(fname)
				if err != nil {
					log.Fatalf("can't stat file: %v", err)
				}
				if st.Size() >= 976560 {
					fname, err = convertToJPG(fname)
					if err != nil {
						log.Fatalf("can't convert to jpeg: %v", err)
					}
				}

				fin, err := os.Open(fname)
				if err != nil {
					log.Fatalf("can't open file: %v", err)
				}

				resp, err := atproto.RepoUploadBlob(ctx, agent.Client(), fin)
				if err != nil {
					log.Fatalf("can't upload blob: %v", err)
				}

				blobs = append(blobs, lexutil.LexBlob{
					Ref:      resp.Blob.Ref,
					MimeType: resp.Blob.MimeType,
					Size:     resp.Blob.Size,
				})
				images = append(images, bskybot.Image{})
			}

			pb = pb.WithImages(blobs, images)
		}

		post, err := pb.Build()
		if err != nil {
			log.Fatalf("can't build post: %v", err)
		}

		cid, uri, err := agent.PostToFeed(ctx, post)
		if err != nil {
			log.Fatalf("can't post to feed: %v", err)
		}

		pathSp := strings.Split(uri, "/")
		id := pathSp[len(pathSp)-1]

		postURL := fmt.Sprintf("https://bsky.app/profile/%s/post/%s", *bskyHandle, id)

		slog.Info("reposted old post", "at", obj.Published, "cid", cid, "uri", uri, "url", postURL)
	}
}

type OrderedCollection struct {
	TotalItems   int    `json:"totalItems"`
	OrderedItems []Item `json:"orderedItems"`
}

type Item struct {
	Type   string          `json:"type"`
	Object json.RawMessage `json:"object"`
}

type Object struct {
	ID            string       `json:"id"`
	To            []string     `json:"to"`
	CC            []string     `json:"cc"`
	InReplyTo     *string      `json:"inReplyTo"`
	Published     time.Time    `json:"published"`
	Content       string       `json:"content"`
	Attachment    []Attachment `json:"attachment"`
	DirectMessage bool         `json:"directMessage"`
}

type Attachment struct {
	MediaType string `json:"mediaType"`
	URL       string `json:"url"`
	Name      string `json:"name"`
}

func convertToJPG(fname string) (string, error) {
	fin, err := os.Open(fname)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %w", fname, err)
	}
	defer fin.Close()

	img, _, err := image.Decode(fin)
	if err != nil {
		return "", fmt.Errorf("failed to decode PNG file: %w", err)
	}

	outFileName := fname[:len(fname)-len(filepath.Ext(fname))] + ".jpeg"
	outFile, err := os.Create(outFileName)
	if err != nil {
		return "", fmt.Errorf("failed to create JPEG file: %w", err)
	}
	defer outFile.Close()

	options := jpeg.Options{Quality: 85}
	if err := jpeg.Encode(outFile, img, &options); err != nil {
		return "", fmt.Errorf("failed to encode JPEG file: %w", err)
	}

	return outFileName, nil
}
