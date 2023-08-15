package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
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

type apiRequestOptions struct {
	payload     []byte
	queryString string
}

type responseFn func([]byte) []byte

type errorResponseFn func([]byte) error

// handleHelpCommand - dibadaba
func handleHelpCommand(_ []string, user *User) error {
	var response []helpDTO
	response = append(response, helpDTO{Desc: "This command", Name: CommandHelp})
	response = append(response, helpDTO{Desc: "What channel are you on.", Name: CommandWhereAmI})
	response = append(response, helpDTO{Desc: "Logged in users on this channel", Name: CommandWho})
	response = append(response, helpDTO{Desc: "For channel operations. Available parameters are 'invite <channelName> <email>', 'create <channelName>.', 'default' that sets the current channel as your default, 'join <channelName>' and 'list'", Name: CommandChannel})
	response = append(response, helpDTO{Desc: "Change your name. Nickname is only persistent if you are registered and logged in. Parameters: <newName>'", Name: CommandNameChange})
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return err
	}
	sendSystemMessage(string(jsonResponse), user, EventHelp)
	return nil
}

func handleWhereCommand(_ []string, user *User) error {
	if len(user.Token) > 0 {
		var currentChannelId string
		if user.CurrentChannelId == "" {
			currentChannelId = PublicChannelName
		} else {
			currentChannelId = user.CurrentChannelId
		}
		sendSystemMessage("You are currently on channel '"+currentChannelId+"'", user, EventNotification)
	} else {
		return errors.New("must be logged in for that command to work")
	}
	return nil
}

func changeNameRequest(user *User, method string, body string, expectedErrorCallback errorResponseFn) error {
	jsonResponse, _ := json.Marshal(nameChangeDTO{Username: body, CreatorToken: user.Token})
	_, err := apiRequest(method, apiRequestOptions{payload: jsonResponse}, "CHAT_CHANGE_NICKNAME", func(response []byte) []byte {
		originalName := user.Name
		user.Name = body
		Users.Store(user, user)
		sendSystemMessage(body, user, EventNameChange)
		sendToOtherOnChannel(originalName+" is now called "+body, user, EventNotification, false, false)
		return nil
	}, expectedErrorCallback)
	return err
}

func handleNameChangeCommand(splitBody []string, user *User) error {
	if len(splitBody) < 2 {
		return nil
	}
	var body = splitBody[1]
	var err error

	if len(body) <= 64 && len(body) >= 1 {
		body = strings.ReplaceAll(body, " ", "")
		if body == "" {
			return errors.New("no empty names")
		}
		key, _ := Users.Load(user)
		user := key.(*User)
		log.Println("handleNameChangeCommand(): User " + user.Name + " is changing name.")
		if user.Name == body {
			return errors.New("you already have that nickname")
		}
		if len(user.Token) > 0 {
			err = changeNameRequest(user, "PUT", body, func(response []byte) error {
				return errors.New("names must be unique")
			})
		} else {
			err = changeNameRequest(user, "POST", body, func(response []byte) error {
				return errors.New("name reserved by registered user. Register to reserve nicknames")
			})
		}
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä nimestä. Lisää vaikka joku error-tyyppi.
		log.Println("New name is too long or too short")
	}
	return err
}

// handleChannelCommand - dibadaba
func handleChannelCommand(commands []string, user *User) error {
	if len(commands) >= 2 {
		var subCommand = commands[1]
		if len(user.Token) > 0 {
			if subCommand == "create" {
				if len(commands) < 3 {
					return errors.New("no empty names")
				}
				var parameter1 = commands[2]
				var parameter2 = false
				if len(commands) >= 4 && commands[3] == "private" {
					parameter2 = true
				}
				parameter1 = strings.ReplaceAll(parameter1, " ", "")
				if parameter1 == "" {
					sendSystemMessage("No empty names!", user, EventErrorNotification)
					return nil
				}
				if parameter1 == PublicChannelName {
					sendSystemMessage("That is a reserved name. Try a different name for your channel.", user, EventErrorNotification)
				}
				if len(parameter1) <= 16 {
					client := &http.Client{}
					jsonResponse, _ := json.Marshal(channelDTO{Name: parameter1, CreatorToken: user.Token, Private: parameter2})
					req, _ := http.NewRequest("POST", os.Getenv("CHAT_CHANNEL_URL"), bytes.NewBuffer(jsonResponse))
					req.Header.Add("Content-Type", "application/json")
					req.Header.Add("Authorization", `Basic `+
						base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
					channelResponse, err := client.Do(req)
					if err != nil || (channelResponse != nil && channelResponse.Status != "200 OK") {
						log.Print("handleChannelCommand():", "Error response "+channelResponse.Status)
						sendSystemMessage("Error creating channel.", user, EventErrorNotification)
					} else {
						sendSystemMessage("Successfully created channel: '"+parameter1+"'. private: "+strconv.FormatBool(parameter2), user, EventNotification)
					}
					defer channelResponse.Body.Close()
				}
			} else if subCommand == "invite" {
				if len(commands) != 4 {
					sendSystemMessage("Insufficient parameters.", user, EventErrorNotification)
					return nil
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
					log.Print("handleChannelCommand():", err)
					// Palauta joku virhe
				}
				if channelResponse != nil && channelResponse.Status != "200 OK" {
					log.Print("handleChannelCommand():", "Error response "+channelResponse.Status)
				}
				defer channelResponse.Body.Close()
				sendSystemMessage("Invite sent successfully to: "+parameter2, user, EventNotification)
			} else if subCommand == "join" {
				if len(commands) != 3 {
					notEnoughParameters(user)
					return nil
				}
				var parameter1 = commands[2]
				var readResponse channelReadResponse
				sendToOtherOnChannel(user.Name+" went looking for better content.", user, EventNotification, false, false)
				client := &http.Client{}
				if len(parameter1) < 1 || parameter1 == PublicChannelName {
					user.CurrentChannelId = ""
					parameter1 = PublicChannelName
				} else {
					jsonResponse, err := json.Marshal(channelGenericDTO{CreatorToken: user.Token, ChannelId: parameter1})
					if err != nil {
						log.Print("handleChannelCommand():", err)
						sendSystemMessage("Error joining channel: '"+parameter1+"'", user, EventErrorNotification)
						return nil
					}
					req, err := http.NewRequest("POST", os.Getenv("CHAT_CHANNEL_LIST_URL"), bytes.NewBuffer(jsonResponse))
					if err != nil {
						log.Print("handleChannelCommand():", err)
						sendSystemMessage("Error joining channel: '"+parameter1+"'", user, EventErrorNotification)
						return nil
					}
					req.Header.Add("Content-Type", "application/json")
					req.Header.Add("Authorization", `Basic `+
						base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
					channelResponse, err := client.Do(req)
					if err != nil {
						log.Print("handleChannelCommand():", err)
						sendSystemMessage("Error joining channel: '"+parameter1+"'", user, EventErrorNotification)
						return nil
					}
					if channelResponse != nil && channelResponse.Status != "200 OK" {
						log.Print("handleChannelCommand():", "Error response "+channelResponse.Status)
						sendSystemMessage("Error joining channel: '"+parameter1+"'", user, EventErrorNotification)
						return nil
					}
					defer channelResponse.Body.Close()
					body, err := ioutil.ReadAll(channelResponse.Body)
					if err != nil {
						log.Print("handleChannelCommand():", err)
						return nil
					}
					_ = json.Unmarshal(body, &readResponse)
					user.CurrentChannelId = readResponse.Name
				}
				err := handleJoin(user)
				if err != nil {
					log.Print("handleChannelCommand():", err)
					sendSystemMessage("Error joining channel", user, EventErrorNotification)
					return nil
				}
				sendSystemMessage("Succesfully joined channel '"+parameter1+"'", user, EventNotification)
			} else if subCommand == "list" {
				client := &http.Client{}
				jsonResponse, _ := json.Marshal(channelGenericDTO{CreatorToken: user.Token})
				req, err := http.NewRequest("POST", os.Getenv("CHAT_CHANNEL_LIST_URL"), bytes.NewBuffer(jsonResponse))
				if err != nil {
					log.Print("handleChannelCommand():", err)
					// Palauta joku virhe
				}
				req.Header.Add("Content-Type", "application/json")
				req.Header.Add("Authorization", `Basic `+
					base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
				channelResponse, err := client.Do(req)
				if err != nil {
					log.Print("handleChannelCommand():", err)
					// Palauta joku virhe
				}
				body, err := ioutil.ReadAll(channelResponse.Body)
				if err != nil {
					log.Print("handleChannelCommand():", err)
					// Palauta joku virhe
				}
				if channelResponse != nil && channelResponse.Status != "200 OK" {
					log.Print("handleChannelCommand():", "Error response "+channelResponse.Status)
				}
				sendSystemMessage(string(body), user, EventChannelList)
				defer channelResponse.Body.Close()
			} else if subCommand == "default" {
				if len(user.CurrentChannelId) == 0 {
					sendSystemMessage("You are currently on the 'public' channel which does not need to be set as default.", user, EventErrorNotification)
					return nil
				}
				client := &http.Client{}
				jsonResponse, _ := json.Marshal(channelGenericDTO{CreatorToken: user.Token, ChannelId: user.CurrentChannelId})
				req, _ := http.NewRequest("PUT", os.Getenv("CHAT_CHANNEL_DEFAULT_URL"), bytes.NewBuffer(jsonResponse))
				req.Header.Add("Content-Type", "application/json")
				req.Header.Add("Authorization", `Basic `+
					base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
				channelResponse, err := client.Do(req)
				if err != nil {
					log.Print("handleChannelCommand():", err)
					// Palauta joku virhe
				}
				if channelResponse != nil && channelResponse.Status != "200 OK" {
					log.Print("handleChannelCommand():", "Error response "+channelResponse.Status)
				}
				sendSystemMessage("Successfully set channel: '"+user.CurrentChannelId+"' as your default channel.", user, EventNotification)
				defer channelResponse.Body.Close()
			}
		} else {
			replyMustBeLoggedIn(user)
		}
	} else {
		notEnoughParameters(user)
	}
	return nil
}

// HandleUserCommand - sdgsdfg
func HandleUserCommand(commands []string, connection *websocket.Conn) {

}

// handleWhoCommand - who is present in the current channel
func handleWhoCommand(_ []string, user *User) error {
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
		log.Printf("handleWhoCommand(): ")
		log.Println(err)
		return err
	}
	sendSystemMessage(string(jsonResponse), user, EventWho)
	return nil
}
