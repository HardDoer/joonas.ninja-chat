package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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

type nameChangeDTO struct {
	Username     string `json:"username"`
	CreatorToken string `json:"creatorToken"`
}

type channelInviteDTO struct {
	Name         string `json:"name"`
	CreatorToken string `json:"creatorToken"`
	NewUser      string `json:"newUser"`
}

type channelGenericDTO struct {
	CreatorToken string `json:"creatorToken"`
	ChannelId    string `json:"channelId"`
}

type channelReadResponse struct {
	Name    string `json:"name"`
	Private bool   `json:"private"`
	Admin   string `json:"admin"`
}

// HandleHelpCommand - dibadaba
func HandleHelpCommand(user *User) {
	var response []helpDTO
	response = append(response, helpDTO{Desc: "This command", Name: CommandHelp})
	response = append(response, helpDTO{Desc: "Logged in users on this channel", Name: CommandWho})
	response = append(response, helpDTO{Desc: "For channel operations. Available parameters are 'invite <channelName> <email>', 'create <channelName>.', 'join <channelName>' and 'list'", Name: CommandChannel})
	response = append(response, helpDTO{Desc: "Change your name. Nickname is only persistent if you are registered and logged in. Parameters: <newName>'", Name: CommandNameChange})
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return
	}
	SendToOne(string(jsonResponse), user, EventHelp)
}

func HandleNameChangeCommand(splitBody []string, user *User) error {
	if len(splitBody) < 2 {
		return nil
	}
	var body = splitBody[1]

	if len(body) <= 64 && len(body) >= 1 {
		var originalName string
		body = strings.ReplaceAll(body, " ", "")
		if body == "" {
			SendToOne("No empty names!", user, EventNotification)
			return nil
		}
		key, _ := Users.Load(user)
		user := key.(*User)
		log.Println("HandleNameChangeCommand(): User " + user.Name + " is changing name.")
		if user.Name == body {
			SendToOne("You already have that nickname.", user, EventNotification)
			return nil
		}
		if len(user.Token) > 0 {
			client := &http.Client{}
			jsonResponse, _ := json.Marshal(nameChangeDTO{Username: body, CreatorToken: user.Token})
			req, _ := http.NewRequest("PUT", os.Getenv("CHAT_CHANGE_NICKNAME"), bytes.NewBuffer(jsonResponse))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", `Basic `+
				base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
			changeNameResponse, err := client.Do(req)
			if err != nil {
				log.Print("HandleNameChangeCommand():", err)
				return err
			}
			if changeNameResponse != nil && changeNameResponse.Status != "200 OK" {
				log.Print("HandleNameChangeCommand():", "Error response "+changeNameResponse.Status)
				SendToOne("Names must be unique.", user, EventNotification)
				return nil
			}
			defer changeNameResponse.Body.Close()
		} else {
			client := &http.Client{}
			jsonResponse, _ := json.Marshal(nameChangeDTO{Username: body})
			req, _ := http.NewRequest("POST", os.Getenv("CHAT_CHECK_NICKNAME"), bytes.NewBuffer(jsonResponse))
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", `Basic `+
				base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
			changeNameResponse, err := client.Do(req)
			if err != nil {
				log.Print("HandleNameChangeCommand():", err)
				return err
			}
			if changeNameResponse != nil && changeNameResponse.Status != "200 OK" {
				log.Print("HandleNameChangeCommand():", "Error response "+changeNameResponse.Status)
				SendToOne("Name reserved by registered user. Register to reserve nicknames.", user, EventNotification)
				return nil
			}
			defer changeNameResponse.Body.Close()
		}
		originalName = user.Name
		user.Name = body
		Users.Store(user, user)
		SendToOne(body, user, EventNameChange)
		SendToOther(originalName+" is now called "+body, user, EventNotification)
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä nimestä. Lisää vaikka joku error-tyyppi.
		log.Println("New name is too long or too short")
	}
	return nil
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
				parameter1 = strings.ReplaceAll(parameter1, " ", "")
				if parameter1 == "" {
					SendToOne("No empty names!", user, EventNotification)
					return
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
					SendToOne("Successfully created channel: '"+parameter1+"'. private: "+strconv.FormatBool(parameter2), user, EventNotification)
				}
			} else if subCommand == "invite" {
				if len(commands) != 4 {
					SendToOne("Insufficient parameters.", user, EventNotification)
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
				SendToOne("Invite sent successfully to: "+parameter2, user, EventNotification)
			} else if subCommand == "join" {
				if len(commands) != 3 {
					return
				}
				var parameter1 = commands[2]
				var readResponse channelReadResponse
				client := &http.Client{}
				jsonResponse, _ := json.Marshal(channelGenericDTO{CreatorToken: user.Token, ChannelId: parameter1})
				req, _ := http.NewRequest("POST", os.Getenv("CHAT_CHANNEL_LIST_URL"), bytes.NewBuffer(jsonResponse))
				req.Header.Add("Content-Type", "application/json")
				req.Header.Add("Authorization", `Basic `+
					base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
				channelResponse, err := client.Do(req)
				if err != nil {
					log.Print("HandleChannelCommand():", err)
					SendToOne("Error joining channel: '"+parameter1+"'", user, EventNotification)
					return
				}
				if channelResponse != nil && channelResponse.Status != "200 OK" {
					log.Print("HandleChannelCommand():", "Error response "+channelResponse.Status)
					SendToOne("Error joining channel: '"+parameter1+"'", user, EventNotification)
					return
				}
				defer channelResponse.Body.Close()
				body, err := ioutil.ReadAll(channelResponse.Body)
				if err != nil {
					log.Print("HandleChannelCommand():", err)
					return
				}
				_ = json.Unmarshal(body, &readResponse)
				SendToOther(user.Name+" went looking for better content.", user, EventNotification)
				user.CurrentChannelId = readResponse.Name
				err = HandleJoin(user)
				SendToOne("Succesfully joined channel '"+parameter1+"'", user, EventNotification)
			} else if subCommand == "list" {
				client := &http.Client{}
				jsonResponse, _ := json.Marshal(channelGenericDTO{CreatorToken: user.Token})
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
			} else if subCommand == "default" {
				if len(user.CurrentChannelId) == 0 {
					SendToOne("You are currently on the 'public' channel which does not need to be set as default.", user, EventNotification)
					return
				}
				client := &http.Client{}
				jsonResponse, _ := json.Marshal(channelGenericDTO{CreatorToken: user.Token, ChannelId: user.CurrentChannelId})
				req, _ := http.NewRequest("PUT", os.Getenv("CHAT_CHANNEL_DEFAULT_URL"), bytes.NewBuffer(jsonResponse))
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
				SendToOne("Successfully set channel: '"+user.CurrentChannelId+"' as your default channel.", user, EventChannelList)
				defer channelResponse.Body.Close()
			}
		} else {
			SendToOne("Must be logged in for that command to work.", user, EventNotification)
		}
	} else {
		SendToOne("Not enough parameters. See '/help'", user, EventNotification)
	}
}

// HandleUserCommand - sdgsdfg
func HandleUserCommand(commands []string, connection *websocket.Conn) {

}

// HandleWhoCommand - who is present in the current channel
func HandleWhoCommand(user *User) {
	var whoIsHere []string
	Users.Range(func(key, value interface{}) bool {
		v := value.(*User)
		if user.CurrentChannelId == v.CurrentChannelId {
			whoIsHere = append(whoIsHere, v.Name)
		}
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
