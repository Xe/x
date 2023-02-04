package mastodon

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"nhooyr.io/websocket"
	"within.website/ln"
	"within.website/ln/opname"
)

// WSSubscribeRequest is a websocket instruction to subscribe to a streaming feed.
type WSSubscribeRequest struct {
	Type    string `json:"type"` // should be "subscribe" or "unsubscribe"
	Stream  string `json:"stream"`
	Hashtag string `json:"hashtag,omitempty"`
}

// WSMessage is a websocket message. Whenever you get something from the streaming service, it will fit into this box.
type WSMessage struct {
	Stream  []string `json:"stream"`
	Event   string   `json:"event"`
	Payload string   `json:"payload"` // json string
}

// StreamMessages is a low-level message streaming facility.
func (c *Client) StreamMessages(ctx context.Context, subreq ...WSSubscribeRequest) (chan WSMessage, error) {
	result := make(chan WSMessage, 10)
	ctx = opname.With(ctx, "websocket-streaming")

	u, err := c.server.Parse("/api/v1/streaming")
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	}

	q := u.Query()
	q.Set("access_token", c.token)
	u.RawQuery = q.Encode()

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := doWebsocket(ctx, u, result, subreq); err != nil {
				ln.Error(ctx, err, ln.Info("websocket error, retrying"))
			}
			time.Sleep(time.Minute)
		}
	}(ctx)

	return result, nil
}

func doWebsocket(ctx context.Context, u *url.URL, result chan WSMessage, subreq []WSSubscribeRequest) error {
	conn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{})
	if err != nil {
		return err
	}
	defer conn.Close(websocket.StatusNormalClosure, "doWebsocket function returned")

	for _, sub := range subreq {
		data, err := json.Marshal(sub)
		if err != nil {
			return err
		}
		err = conn.Write(ctx, websocket.MessageText, data)
		if err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
		}

		msgType, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}

		if msgType != websocket.MessageText {
			ln.Log(ctx, ln.Info("got non-text message from mastodon"))
			continue
		}

		var msg WSMessage
		err = json.Unmarshal(data, &msg)
		if err != nil {
			return err
		}

		result <- msg
	}
}
