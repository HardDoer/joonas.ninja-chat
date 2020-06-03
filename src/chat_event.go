package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"strings"
	"time"
)

// HandleMessageEvent - 
func HandleMessageEvent(body string, user *User) error {
	var senderName = ""
	if len(body) < 512 {
		if strings.Index(body, "/") != 0 {
			value, _ := Users.Load(user)
			user := value.(*User)
			senderName = user.Name
			SendToAll(body, senderName, EventMessage)
		} else {
			handleCommand(body, user)
		}
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä viestistä.
		log.Println("Message is too long")
	}
	return nil
}

// HandleJoin - 
func HandleJoin(chatUser *User) error {
	response := EventData{Event: EventJoin, Body: chatUser.Name, UserCount: userCount, CreatedDate: time.Now()}
	chatHistory := GetChatHistory()
	if chatHistory != nil {
		if err := chatUser.write(websocket.TextMessage, chatHistory); err != nil {
			return err
		}
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return err
	}
	if err := chatUser.write(websocket.TextMessage, jsonResponse); err != nil {
		return err
	}
	SendToOther(chatUser.Name+" has joined the chat.", chatUser, EventNotification)
	return nil
}

// HandleTypingEvent - 
func HandleTypingEvent(body string, user *User) error {
	return nil
}

// HandleNameChangeEvent - 
func HandleNameChangeEvent(body string, user *User) error {
	if len(body) <= 64 && len(body) >= 1 {
		var originalName string
		body = strings.ReplaceAll(body, " ", "")
		if body == "" {
			// TODO. Palauta joku virhe käyttäjälle vääränlaisesta nimestä.
			log.Println("No empty names!")
			return nil
		}
		key, _ := Users.Load(user)
		user := key.(*User)
		log.Println("handleNameChangeEvent(): User " + user.Name + " is changing name.")
		originalName = user.Name
		user.Name = body
		Users.Store(user, user)
		response := EventData{Event: EventNameChange, Body: user.Name, UserCount: userCount, CreatedDate: time.Now()}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return err
		}
		if err := user.write(websocket.TextMessage, jsonResponse); err != nil {
			return err
		}
		SendToOther(originalName+" is now called "+body, user, EventNotification)
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä nimestä. Lisää vaikka joku error-tyyppi.
		log.Println("New name is too long or too short")
	}
	return nil
}