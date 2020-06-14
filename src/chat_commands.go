package main

import (
	"log"
	"github.com/gorilla/websocket"
	"encoding/json"
)

// HandleChannelCommand - dibadaba
func HandleChannelCommand(commands []string, connection *websocket.Conn) {
	
}

// HandleUserCommand - sdgsdfg
func HandleUserCommand(commands []string, connection *websocket.Conn) {
	
}

// HandleWhoCommand - who is present in the current channel
func HandleWhoCommand(user *User) {
	var whoIsHere []string
	Users.Range(func(key, value interface{}) bool {
		v := value.(*User)
		whoIsHere = append(whoIsHere, v.Name)
		return true
	})
	jsonResponse, err := json.Marshal(whoIsHere)
	if err != nil {
		log.Printf("HandleWhoCommand(): ")
		log.Println(err)
		return
	}
	SendToOne(string(jsonResponse), user, EventWho)
}
