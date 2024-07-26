package algora

import "github.com/gorilla/websocket"

type ChatConn struct {
	*websocket.Conn
}
