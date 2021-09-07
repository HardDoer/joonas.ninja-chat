package main

import (
	"github.com/gorilla/websocket"
	"sync"
	"time"
)

// User - A chat user.
type User struct {
	Name       string
	Token      string
	Connection *websocket.Conn
	mutex      sync.Mutex
}

func (u *User) write(messageType int, data []byte) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	u.Connection.SetWriteDeadline(time.Now().Add(pingWait))
	err := u.Connection.WriteMessage(messageType, data)
	return err
}
