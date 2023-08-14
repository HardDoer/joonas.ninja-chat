package main

import (
	"log"
	"strings"
	"time"
	"github.com/gorilla/websocket"
)

type chatLogin struct {
	Scope     string `json:"scope"`
	GrantType string `json:"grant_type"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type loginDTO struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type gatewayDTO struct {
	Token string `json:"token"`
}

type tokenValidationRes struct {
	Username       string `json:"username"`
	DefaultChannel string `json:"defaultChannel"`
}

func handleCommand(body string, user *User) {
	var splitBody = strings.Split(body, "/")
	splitBody = strings.Split(splitBody[1], " ")
	command := splitBody[0]
	// TODO. Mappiin nämä ja errori palautetaan.
	switch command {
	case CommandWho:
		handleWhoCommand(user)
	case CommandNameChange:
		handleNameChangeCommand(splitBody, user)
	case CommandHelp:
		handleHelpCommand(user)
	case CommandChannel:
		handleChannelCommand(splitBody, user)
	case CommandWhereAmI:
		handleWhereCommand(user)
	default:
		sendToOne("Command not recognized. Type '/help' for list of chat commands.", user, EventErrorNotification)
	}
}

// handleMessageEvent -
func handleMessageEvent(body string, user *User) error {
	var senderName = ""
	if len(body) < 4096 {
		if strings.Index(body, "/") != 0 {
			value, _ := Users.Load(user)
			user := value.(*User)
			senderName = user.Name
			sendToAll(body, user.CurrentChannelId, senderName, EventMessage, true)
		} else {
			handleCommand(body, user)
		}
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä viestistä.
		log.Println("Message is too long")
	}
	return nil
}

// handleJoin -
func handleJoin(chatUser *User) error {
	chatHistory := getChatHistory(chatUser.CurrentChannelId)
	if chatHistory != nil {
		if err := chatUser.write(websocket.TextMessage, chatHistory); err != nil {
			return err
		}
	} else {
		sendToOne("Error refreshing chat history.", chatUser, EventErrorNotification)
	}
	response := EventData{Event: EventJoin, ChannelId: chatUser.CurrentChannelId, Body: chatUser.Name, UserCount: UserCount, CreatedDate: time.Now()}
	jsonResponse, err := marshalJson(response)
	if err != nil {
		return err
	}
	if err := chatUser.write(websocket.TextMessage, jsonResponse); err != nil {
		return err
	}
	sendToOtherOnChannel(chatUser.Name+" has joined the channel.", chatUser, EventNotification)
	return nil
}

// HandleTypingEvent -
func HandleTypingEvent(body string, user *User) error {
	return nil
}
