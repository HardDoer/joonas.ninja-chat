package util

import (
	"github.com/gorilla/websocket"
)

// User - A chat user.
type User struct {
	Name       string
	Connection *websocket.Conn
}

// EventData - A data structure that contains information about the current chat event.
type EventData struct {
	Event string `json:"event"`
	Body  string `json:"body"`
}
