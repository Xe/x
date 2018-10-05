/*
Copyright 2017-2018 Mikael Berthe

Licensed under the MIT license.  Please see the LICENSE file is this directory.
*/

package madon

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// StreamEvent contains a single event from the streaming API
type StreamEvent struct {
	Event string      // Name of the event (error, update, notification or delete)
	Data  interface{} // Status, Notification or status ID
	Error error       // Error message from the StreamListener
}

// openStream opens a stream URL and returns an http.Response
// Note that the caller should close the connection when it's done reading
// the stream.
// The stream name can be "user", "local", "public", "list" or "hashtag".
// When it is "hashtag", the param argument contains the hashtag.
// When it is "list", the param argument contains the list ID.
func (mc *Client) openStream(streamName, param string) (*websocket.Conn, error) {
	var tag, list string

	switch streamName {
	case "user", "public", "public:local":
	case "hashtag":
		if param == "" {
			return nil, ErrInvalidParameter
		}
		tag = param
	case "list":
		if param == "" {
			return nil, ErrInvalidParameter
		}
		list = param
	default:
		return nil, ErrInvalidParameter
	}

	if !strings.HasPrefix(mc.APIBase, "http") {
		return nil, errors.New("cannot create Websocket URL: unexpected API base URL")
	}

	// Build streaming websocket URL
	u, err := url.Parse("ws" + mc.APIBase[4:] + "/streaming/")
	if err != nil {
		return nil, errors.Wrap(err, "cannot create Websocket URL")
	}

	urlParams := url.Values{}
	urlParams.Add("stream", streamName)
	urlParams.Add("access_token", mc.UserToken.AccessToken)
	if tag != "" {
		urlParams.Add("tag", tag)
	} else if list != "" {
		urlParams.Add("list", list)
	}
	u.RawQuery = urlParams.Encode()

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	return c, err
}

// readStream reads from the http.Response and sends events to the events channel
// It stops when the connection is closed or when the stopCh channel is closed.
// The foroutine will close the doneCh channel when it terminates.
func (mc *Client) readStream(events chan<- StreamEvent, stopCh <-chan bool, doneCh chan bool, c *websocket.Conn) {
	defer c.Close()
	defer close(doneCh)

	go func() {
		select {
		case <-stopCh:
			// Close connection
			c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		case <-doneCh:
			// Leave
		}
	}()

	for {
		var msg struct {
			Event   string
			Payload interface{}
		}

		err := c.ReadJSON(&msg)
		if err != nil {
			if strings.Contains(err.Error(), "close 1000 (normal)") {
				break // Connection properly closed
			}
			e := errors.Wrap(err, "read error")
			events <- StreamEvent{Event: "error", Error: e}
			break
		}

		var obj interface{}

		// Decode API object
		switch msg.Event {
		case "update":
			strPayload, ok := msg.Payload.(string)
			if !ok {
				e := errors.New("could not decode status: payload isn't a string")
				events <- StreamEvent{Event: "error", Error: e}
				continue
			}
			var s Status
			if err := json.Unmarshal([]byte(strPayload), &s); err != nil {
				e := errors.Wrap(err, "could not decode status")
				events <- StreamEvent{Event: "error", Error: e}
				continue
			}
			obj = s
		case "notification":
			strPayload, ok := msg.Payload.(string)
			if !ok {
				e := errors.New("could not decode notification: payload isn't a string")
				events <- StreamEvent{Event: "error", Error: e}
				continue
			}
			var notif Notification
			if err := json.Unmarshal([]byte(strPayload), &notif); err != nil {
				e := errors.Wrap(err, "could not decode notification")
				events <- StreamEvent{Event: "error", Error: e}
				continue
			}
			obj = notif
		case "delete":
			strPayload, ok := msg.Payload.(string)
			if !ok {
				e := errors.New("could not decode deletion: payload isn't a string")
				events <- StreamEvent{Event: "error", Error: e}
				continue
			}
			obj = strPayload // statusID
		default:
			e := errors.Errorf("unhandled event '%s'", msg.Event)
			events <- StreamEvent{Event: "error", Error: e}
			continue
		}

		// Send event to the channel
		events <- StreamEvent{Event: msg.Event, Data: obj}
	}
}

// StreamListener listens to a stream from the Mastodon server
// The stream 'name' can be "user", "local", "public" or "hashtag".
// For 'hashtag', the hashTag argument cannot be empty.
// The events are sent to the events channel (the errors as well).
// The streaming is terminated if the 'stopCh' channel is closed.
// The 'doneCh' channel is closed if the connection is closed by the server.
// Please note that this method launches a goroutine to listen to the events.
func (mc *Client) StreamListener(name, hashTag string, events chan<- StreamEvent, stopCh <-chan bool, doneCh chan bool) error {
	if mc == nil {
		return ErrUninitializedClient
	}

	conn, err := mc.openStream(name, hashTag)
	if err != nil {
		return err
	}
	go mc.readStream(events, stopCh, doneCh, conn)
	return nil
}
