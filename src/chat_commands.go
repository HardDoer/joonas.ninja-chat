package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
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
	Private      bool   `json:"private"`
}

type channelInviteDTO struct {
	Name         string `json:"name"`
	CreatorToken string `json:"creatorToken"`
	NewUser      string `json:"newUser"`
}

type channelListDTO struct {
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
	if len(commands) >= 2 {
		var subCommand = commands[1]
		if len(user.Token) > 0 {
			if subCommand == "create" {
				var parameter1 = commands[2]
				var parameter2 = false
				if len(commands) >= 4 && commands[3] == "private" {
					parameter2 = true
				}
				if len(parameter1) <= 16 {
					client := &http.Client{}
					jsonResponse, _ := json.Marshal(channelDTO{Name: parameter1, CreatorToken: user.Token, Private: parameter2})
					req, _ := http.NewRequest("POST", os.Getenv("CHAT_CHANNEL_URL"), bytes.NewBuffer(jsonResponse))
					req.Header.Add("Content-Type", "application/json")
					req.Header.Add("Authorization", `Basic `+
						base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
					channelResponse, err := client.Do(req)
					if err != nil {
						log.Print("HandleChannelCommand():", err)
						// Palauta joku virhe
					}
					if channelResponse != nil && channelResponse.Status != "200 OK" {
						log.Print("HandleChannelCommand():", "Error response "+channelResponse.Status)
					}
					defer channelResponse.Body.Close()
				}
			} else if subCommand == "invite" {
				if len(commands) != 4 {
					return
				}
				var parameter1 = commands[2]
				var parameter2 = commands[3]
				client := &http.Client{}
				jsonResponse, _ := json.Marshal(channelInviteDTO{Name: parameter1, CreatorToken: user.Token, NewUser: parameter2})
				req, _ := http.NewRequest("PUT", os.Getenv("CHAT_CHANNEL_INVITE_URL"), bytes.NewBuffer(jsonResponse))
				req.Header.Add("Content-Type", "application/json")
				req.Header.Add("Authorization", `Basic `+
					base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
				channelResponse, err := client.Do(req)
				if err != nil {
					log.Print("HandleChannelCommand():", err)
					// Palauta joku virhe
				}
				if channelResponse != nil && channelResponse.Status != "200 OK" {
					log.Print("HandleChannelCommand():", "Error response "+channelResponse.Status)
				}
				defer channelResponse.Body.Close()
			} else if subCommand == "list" {
				client := &http.Client{}
				jsonResponse, _ := json.Marshal(channelListDTO{CreatorToken: user.Token})
				req, _ := http.NewRequest("POST", os.Getenv("CHAT_CHANNEL_LIST_URL"), bytes.NewBuffer(jsonResponse))
				req.Header.Add("Content-Type", "application/json")
				req.Header.Add("Authorization", `Basic `+
					base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
				channelResponse, err := client.Do(req)
				body, err := ioutil.ReadAll(channelResponse.Body)
				if err != nil {
					log.Print("HandleChannelCommand():", err)
					// Palauta joku virhe
				}
				if channelResponse != nil && channelResponse.Status != "200 OK" {
					log.Print("HandleChannelCommand():", "Error response "+channelResponse.Status)
				}
				SendToOne(string(body), user, EventChannelList)
				defer channelResponse.Body.Close()
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
