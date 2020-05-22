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

// HandleWhoCommand - dibadaba
func HandleWhoCommand(connection *websocket.Conn) {
	var whoIsHere []string
	Users.Range(func(key, value interface{}) bool {
		v := value.(User)
		whoIsHere = append(whoIsHere, v.Name)
		return true
	})
	jsonResponse, err := json.Marshal(whoIsHere)
	if err != nil {
		log.Printf("HandleWhoCommand(): ")
		log.Println(err)
		return
	}
	SendToOne(string(jsonResponse), connection, EventWho)
}
