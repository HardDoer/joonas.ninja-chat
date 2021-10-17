package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

type helpDTO struct {
	Desc string `json:"description"`
	Name string `json:"name"`
}

type channelDTO struct {
	Name         string `json:"name"`
	CreatorToken string `json:"creatorToken"`
}

// HandleHelpCommand - dibadaba
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

// HandleChannelCommand - dibadaba
func HandleChannelCommand(commands []string, user *User) {
	if len(commands) > 2 {
		var subCommand = commands[1]
		var parameter = commands[2]
		if len(user.Token) > 0 {
			if subCommand == "create" && len(parameter) <= 16 {
				client := &http.Client{}
				jsonResponse, _ := json.Marshal(channelDTO{Name: parameter, CreatorToken: user.Token})
				req, _ := http.NewRequest("POST", os.Getenv("CHAT_CHANNEL_URL"), bytes.NewBuffer(jsonResponse))
				req.Header.Add("Content-Type", "application/json")
				req.Header.Add("Authorization", `Basic `+
					base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("GATEWAY_KEY"))))
				channelResponse, err := client.Do(req)
				if err != nil {
					log.Print("HandleChannelCommand():", err)
				}
				if channelResponse != nil && channelResponse.Status != "200 OK" {
					log.Print("HandleChannelCommand():", "Error response "+channelResponse.Status)
				}
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
