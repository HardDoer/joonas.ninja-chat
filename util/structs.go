package util

import (
	"github.com/gorilla/websocket"
)

// User - A chat user.
type User struct {
	Name       string
	Connection *websocket.Conn
}

// ChatRequest - A request you can send to the backend.
type ChatRequest struct {
	Event string
	Body  string
}
