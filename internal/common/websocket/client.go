package websocket

import (
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID            string
	Conn          *websocket.Conn
	Send          chan []byte
	Authenticated bool
	LastPong      time.Time
}
