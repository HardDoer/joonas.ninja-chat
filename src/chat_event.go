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
		sendOneMessage("Command not recognized. Type '/help' for list of chat commands.", user, EventErrorNotification)
	}
}

// handleMessageEvent -
func handleMessageEvent(body string, user *User) error {
	if len(body) < 256 {
		if strings.Index(body, "/") != 0 {
			value, _ := Users.Load(user)
			user := value.(*User)
			sendToAllOnChannel(body, user, EventMessage, true)
		} else {
			handleCommand(body, user)
		}
	} else {
		sendOneMessage("Message is too long.", user, EventErrorNotification)
	}
	return nil
}

// handleJoin -
func handleJoin(chatUser *User) error {
	// TODO. Refaktoroi chat history palauttamaan vaan se rakennettu kikkare. Sit voidaan lähettää se suoraan noiden message funktioiden kautta eikä tarvitse
	// erikseen kirjoittaa sitä tässä.
	chatHistory := getChatHistory(chatUser.CurrentChannelId)
	if chatHistory != nil {
		if err := chatUser.write(websocket.TextMessage, chatHistory); err != nil {
			return err
		}
	} else {
		sendOneMessage("Error refreshing chat history.", chatUser, EventErrorNotification)
	}
	sendOneMessage(chatUser.Name, chatUser, EventJoin)
	sendToOtherOnChannel(chatUser.Name+" has joined the channel.", chatUser, EventNotification, false)
	return nil
}

// HandleTypingEvent -
func HandleTypingEvent(body string, user *User) error {
	return nil
}
