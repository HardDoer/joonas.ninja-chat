package main

import (
	"encoding/json"
	"errors"
	"log"
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
		return replyMustBeLoggedIn()
	}
	return nil
}

func changeNameRequest(user *User, method string, env string, body string) error {
	jsonResponse, _ := json.Marshal(nameChangeDTO{Username: body, CreatorToken: user.Token})
	_, err := apiRequest(method, newApiRequestOptions(&apiRequestOptions{payload: jsonResponse}), env, func(response []byte) []byte {
		originalName := user.Name
		user.Name = body
		Users.Store(user, user)
		sendSystemMessage(body, user, EventNameChange)
		sendToOtherOnChannel(originalName+" is now called "+body, user, EventNotification, false, false)
		return nil
	})
	return err
}

func handleNameChangeCommand(splitBody []string, u *User) error {
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
		key, _ := Users.Load(u)
		user := key.(*User)
		log.Println("handleNameChangeCommand(): User " + user.Name + " is changing name.")
		if user.Name == body {
			return errors.New("you already have that nickname")
		}
		if len(user.Token) > 0 {
			if err := changeNameRequest(user, "PUT", "CHAT_CHANGE_NICKNAME", body); err != nil {
				return errors.New("names must be unique")
			}

		} else {
			if err := changeNameRequest(user, "POST", "CHAT_CHECK_NICKNAME", body); err != nil {
				return errors.New("name reserved by registered user. Register to reserve nicknames")
			}
		}
	} else {
		return errors.New("that name is too long ")
	}
	return err
}

// handleChannelCommand - dibadaba
func handleChannelCommand(commands []string, user *User) error {
	if len(commands) >= 2 {
		var subCommand = commands[1]
		if len(user.Token) > 0 {
			commandFn, ok := getChannelCommand(subCommand)
			if (!ok) {
				return notEnoughParameters()
			}
			return commandFn(commands, user)
		} else {
			return replyMustBeLoggedIn()
		}
	}
	return notEnoughParameters()
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
