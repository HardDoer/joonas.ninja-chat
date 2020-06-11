package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type chatLogin struct {
	Scope     string `json:"scope"`
	GrantType string `json:"grant_type"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type gatewayResponse struct {
	Token string `json:"token"`
}

func handleCommand(body string, user *User) {
	var splitBody = strings.Split(body, "/")
	splitBody = strings.Split(splitBody[1], " ")
	command := splitBody[0]
	switch command {
	case CommandWho:
		HandleWhoCommand(user)
		/*
			case CommandChannel:
				HandleChannelCommand(splitBody, connection)
		*/
	default:
		SendToOne("Command "+"'"+body+"' not recognized.", user, EventNotification)
	}
}

func loginRequest(email string, password string) (res gatewayResponse, err error) {
	var gatewayRes gatewayResponse
	chatLoginRequest := chatLogin{Scope: "chat", GrantType: "client_credentials", Email: email, Password: password}
	jsonResponse, err := json.Marshal(chatLoginRequest)
	if err != nil {
		log.Print("loginRequest():", err)
		return gatewayRes, err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", os.Getenv("CHAT_LOGIN_URL"), bytes.NewBuffer(jsonResponse))
	if err != nil {
		log.Print("loginRequest():", err)
		return gatewayRes, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", `Basic `+
		base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("GATEWAY_KEY"))))
	loginResponse, err := client.Do(req)
	if err != nil {
		log.Print("loginRequest():", err)
		return gatewayRes, err
	}
	if loginResponse != nil && loginResponse.Status != "200 OK" {
		log.Print("loginRequest():", "Error response "+loginResponse.Status)
		return gatewayRes, errors.New("Error response " + loginResponse.Status)
	}
	defer loginResponse.Body.Close()
	body, err := ioutil.ReadAll(loginResponse.Body)
	if err != nil {
		log.Print("loginRequest():", err)
		return gatewayRes, err
	}
	err = json.Unmarshal(body, &gatewayRes)
	if err != nil {
		log.Print("loginRequest():", err)
		return gatewayRes, err
	}
	return gatewayRes, nil
}

// HandleLoginEvent - Handles the logic with user login.
func HandleLoginEvent(body string, user *User) error {
	var email string
	var password string
	var parsedBody []string
	var marshalAndWrite = func(data EventData) error {
		jsonResponse, err := json.Marshal(data)
		if err != nil {
			log.Print("HandleLoginEvent():", err)
		}
		if err := user.write(websocket.TextMessage, jsonResponse); err != nil {
			log.Print("HandleLoginEvent():", err)
			return err
		}
		return nil
	}

	if len(body) < 512 {
		parsedBody = strings.Split(body, ":")
		email = parsedBody[0]
		password = parsedBody[1]
		if len(email) > 1 && len(password) > 1 {
			loginRes, loginError := loginRequest(email, password)
			if loginError != nil {
				response := EventData{Event: EventNotification, Body: "Login error.", UserCount: UserCount, CreatedDate: time.Now()}
				return marshalAndWrite(response)
			}
			response := EventData{Event: EventLogin, Body: loginRes.Token, UserCount: UserCount, CreatedDate: time.Now()}
			log.Print("HandleLoginEvent():", "Login successful")
			return marshalAndWrite(response)
		}
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä viestistä.
		log.Println("Message is too long")
	}
	return nil
}

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
	response := EventData{Event: EventJoin, Body: chatUser.Name, UserCount: UserCount, CreatedDate: time.Now()}
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
		response := EventData{Event: EventNameChange, Body: user.Name, UserCount: UserCount, CreatedDate: time.Now()}
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
