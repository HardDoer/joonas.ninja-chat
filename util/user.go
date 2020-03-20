package util;

import (
	"github.com/gorilla/websocket"
)

// User - A chat user.
type User struct {
	Name string;
	Connection *websocket.Conn;
}