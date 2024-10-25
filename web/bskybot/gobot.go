package bskybot

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	appbsky "github.com/bluesky-social/indigo/api/bsky"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/xrpc"
)

const defaultPDS = "https://bsky.social"

var blob []lexutil.LexBlob

// Wrapper over the atproto xrpc transport
type BskyAgent struct {
	// xrpc transport, a wrapper around http server
	client *xrpc.Client
	handle string
	apikey string
	t      *time.Ticker
	lock   sync.Mutex
}

// Creates new BlueSky Agent
func NewAgent(ctx context.Context, server string, handle string, apikey string) BskyAgent {
	if server == "" {
		server = defaultPDS
	}

	return BskyAgent{
		client: &xrpc.Client{
			Client: new(http.Client),
			Host:   server,
		},
		handle: handle,
		apikey: apikey,
	}
}

// Connect and Authenticate to the provided Personal Data Server, default is Bluesky PDS
// No need to refresh the access token if the bot script will be executed based on the cron job
func (c *BskyAgent) Connect(ctx context.Context) error {
	input_for_session := &atproto.ServerCreateSession_Input{
		Identifier: c.handle,
		Password:   c.apikey,
	}

	session, err := atproto.ServerCreateSession(ctx, c.client, input_for_session)
	if err != nil {
		return fmt.Errorf("UNABLE TO CONNECT: %v", err)
	}

	// Access Token is used to make authenticated requests
	// Refresh Token allows to generate a new Access Token
	c.client.Auth = &xrpc.AuthInfo{
		AccessJwt:  session.AccessJwt,
		RefreshJwt: session.RefreshJwt,
		Handle:     session.Handle,
		Did:        session.Did,
	}

	slog.Debug("authed", "did", session.Did, "handle", session.Handle)

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.t == nil {
		return nil
	}

	t := time.NewTicker(10 * time.Minute)
	c.t = t

	go c.reauthLoop()

	return nil
}

func (c *BskyAgent) reauthLoop() {
	slog.Debug("started auth background process")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for range c.t.C {
		if err := c.refreshAuth(ctx); err != nil {
			slog.Error("can't refresh auth, auth may be bad?", "err", err)
			continue
		}
	}
}

func (c *BskyAgent) refreshAuth(ctx context.Context) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	resp, err := atproto.ServerRefreshSession(ctx, c.client)
	if err != nil {
		return err
	}

	c.client.Auth = &xrpc.AuthInfo{
		AccessJwt:  resp.AccessJwt,
		RefreshJwt: resp.RefreshJwt,
		Handle:     resp.Handle,
		Did:        resp.Did,
	}

	return nil
}

func (c *BskyAgent) UploadImages(ctx context.Context, images ...Image) ([]lexutil.LexBlob, error) {
	for _, img := range images {
		getImage, err := getImageAsBuffer(img.Uri.String())
		if err != nil {
			log.Printf("Couldn't retrive the image: %v , %v", img, err)
		}

		resp, err := atproto.RepoUploadBlob(ctx, c.client, bytes.NewReader(getImage))
		if err != nil {
			return nil, err
		}

		blob = append(blob, lexutil.LexBlob{
			Ref:      resp.Blob.Ref,
			MimeType: resp.Blob.MimeType,
			Size:     resp.Blob.Size,
		})
	}
	return blob, nil
}

// Post to social app
func (c *BskyAgent) PostToFeed(ctx context.Context, post appbsky.FeedPost) (string, string, error) {
	post_input := &atproto.RepoCreateRecord_Input{
		// collection: The NSID of the record collection.
		Collection: "app.bsky.feed.post",
		// repo: The handle or DID of the repo (aka, current account).
		Repo: c.client.Auth.Did,
		// record: The record itself. Must contain a $type field.
		Record: &lexutil.LexiconTypeDecoder{Val: &post},
	}

	response, err := atproto.RepoCreateRecord(ctx, c.client, post_input)
	if err != nil {
		return "", "", fmt.Errorf("unable to post, %v", err)
	}

	return response.Cid, response.Uri, nil
}

func getImageAsBuffer(imageURL string) ([]byte, error) {
	// Fetch image
	response, err := http.Get(imageURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	// Check response status
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch image: %s", response.Status)
	}

	// Read response body
	imageData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return imageData, nil
}
