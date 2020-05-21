package main

import (
	"github.com/gorilla/websocket"
)

// User - A chat user.
type User struct {
	Name       string
	Token      string
	Connection *websocket.Conn
}
