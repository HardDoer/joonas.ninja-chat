package main

import (
	"reflect"
	"strings"
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

func getCommand(command string) (func([]string, *User) error, bool) {
	var commands = map[string]func([]string, *User) error{
		CommandWho: handleWhoCommand,
		CommandNameChange: handleNameChangeCommand,
		CommandHelp: handleHelpCommand,
		CommandChannel: handleChannelCommand,
		CommandWhereAmI: handleWhereCommand,
	}
	commandFn, ok := commands[command]
	return commandFn, ok
}

func handleCommand(body string, user *User) {
	var splitBody = strings.Split(body, "/")
	splitBody = strings.Split(splitBody[1], " ")
	command := splitBody[0]
	// TODO. Mappiin nämä ja errori palautetaan.
	commandFn, ok := getCommand(command)
	if (!ok) {
		sendSystemMessage("Command not recognized. Type '/help' for list of chat commands.", user, EventErrorNotification)
	} else {
		commandFn(splitBody, user)
	}
}

// handleMessageEvent -
func handleMessageEvent(body string, user *User) error {
	if len(body) < 256 {
		if strings.Index(body, "/") != 0 {
			value, _ := Users.Load(user)
			user := value.(*User)
			sendToAllOnChannel(body, user, EventMessage, true, true)
		} else {
			handleCommand(body, user)
		}
	} else {
		sendSystemMessage("Message is too long.", user, EventErrorNotification)
	}
	return nil
}

// handleJoin -
func handleJoin(chatUser *User) error {
	chatHistory := getChatHistory(chatUser.CurrentChannelId)
	if !reflect.DeepEqual(chatHistory, ChatHistory{}) {
		marshalAndWriteToStream(chatUser, chatHistory)
	} else {
		sendSystemMessage("Error refreshing chat history.", chatUser, EventErrorNotification)
	}
	sendSystemMessage(chatUser.Name, chatUser, EventJoin)
	sendToOtherOnChannel(chatUser.Name+" has joined the channel.", chatUser, EventNotification, false, false)
	return nil
}

// HandleTypingEvent -
func handleTypingEvent(body string, user *User) error {
	return nil
}
