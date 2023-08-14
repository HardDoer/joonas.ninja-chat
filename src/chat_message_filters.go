package main

import (
	"log"

	"github.com/gorilla/websocket"
)

// sends the body string data to all connected clients on the same channel
func sendToAllOnChannelFilter(user *User, jsonResponse []byte) func(key any, value any) bool  {
	return func(key, value interface{}) bool {
		var userValue = value.(*User)
		if userValue.CurrentChannelId == user.CurrentChannelId {
			if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
				log.Print("sendToAllOnChannelFilter():", err)
			}
		}
		return true
	}
}
// sends the body string data to all connected clients
func sendToAllFilter(user *User, jsonResponse []byte) func(key any, value any) bool  {
	return func(key, value interface{}) bool {
		var userValue = value.(*User)
		if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
			log.Print("sendToAllFilter():", err)
		}
		return true
	}
}

// sends the body string data to all connected clients on the same channel except the parameter given client
func sendToOtherOnChannelFilter(user *User, jsonResponse []byte) func(key any, value any) bool  {
	return func(key, value interface{}) bool {
		userValue := value.(*User)
		if userValue != user && userValue.CurrentChannelId == user.CurrentChannelId {
			if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
				log.Print("sendToOtherOnChannelFilter():", err)
			}
		}
		return true
	}
}

// sends the body string data to all connected clients except the parameter given client
func sendToOtherEverywhereFilter(user *User, jsonResponse []byte) func(key any, value any) bool {
	return func(key, value interface{}) bool {
		userValue := value.(*User)
		if userValue != user {
			if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
				log.Print("sendToOtherEverywhereFilter():", err)
			}
		}
		return true
	}
}