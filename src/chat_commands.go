package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
)

type helpDTO struct {
	Desc string `json:"description"`
	Name string `json:"name"`
}

// HandleChannelCommand - dibadaba
func HandleHelpCommand(user *User) {
	var response []helpDTO
	response = append(response, helpDTO{Desc: "This command", Name: CommandHelp})
	response = append(response, helpDTO{Desc: "Logged in users on this channel", Name: CommandWho})
	response = append(response, helpDTO{Desc: "For channel operations. Additional parameters are 'join <channelId>' and 'create <channelName>.'", Name: CommandChannel})
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return
	}
	SendToOne(string(jsonResponse), user, EventHelp)
}

func HandleChannelCommand(commands []string, user *User) {
	if (len(commands) > 2) {
		var subCommand = commands[1]
		var parameter = commands[2]
		if len(user.Token) > 0 {
			if (subCommand == "create") {
				// TODO. Luo kanava.
				log.Println(parameter)
			}
		}
	} //TODO. Palauta virheohje.
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
