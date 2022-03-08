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
	Username string `json:"username"`
	DefaultChannel string `json:"defaultChannel"`
}

func handleCommand(body string, user *User) {
	var splitBody = strings.Split(body, "/")
	splitBody = strings.Split(splitBody[1], " ")
	command := splitBody[0]
	switch command {
	case CommandWho:
		HandleWhoCommand(user)
	case CommandNameChange:
		HandleNameChangeCommand(splitBody, user)
	case CommandHelp:
		HandleHelpCommand(user)
	case CommandChannel:
		HandleChannelCommand(splitBody, user)
	case CommandWhereAmI:
		HandleWhereCommand(user)
	default:
		SendToOne("Command not recognized. Type '/help' for list of chat commands.", user, EventErrorNotification)
	}
}

func validateToken(token string) (validationRes tokenValidationRes, err error) {
	chatTokenRequest := gatewayDTO{Token: token}
	jsonResponse, err := json.Marshal(chatTokenRequest)
	var tokenJson tokenValidationRes

	if err != nil {
		log.Print("validateToken():", err)
		return tokenJson, err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", os.Getenv("CHAT_TOKEN_URL"), bytes.NewBuffer(jsonResponse))
	if err != nil {
		log.Print("validateToken():", err)
		return tokenJson, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", `Basic `+
		base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("GATEWAY_KEY"))))
	tokenResponse, err := client.Do(req)
	if err != nil {
		log.Print("validateToken():", err)
		return tokenJson, err
	}
	if tokenResponse != nil && tokenResponse.Status != "200 OK" {
		log.Print("validateToken():", "Error response "+tokenResponse.Status)
		return tokenJson, errors.New("Error response " + tokenResponse.Status)
	}
	defer tokenResponse.Body.Close()
	body, err := ioutil.ReadAll(tokenResponse.Body)
	if err != nil {
		log.Print("getChatHistory():", err)
		return tokenJson, err
	}
	err = json.Unmarshal(body, &tokenJson)
	if err != nil {
		log.Print("getChatHistory():", err)
		return tokenJson, err
	}
	return tokenJson, nil
}

func loginRequest(email string, password string) (res gatewayDTO, err error) {
	var gatewayRes gatewayDTO
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

// HandleMessageEvent -
func HandleMessageEvent(body string, user *User) error {
	var senderName = ""
	if len(body) < 4096 {
		if strings.Index(body, "/") != 0 {
			value, _ := Users.Load(user)
			user := value.(*User)
			senderName = user.Name
			SendToAll(body, user.CurrentChannelId, senderName, EventMessage, true)
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
	response := EventData{Event: EventJoin, ChannelId: chatUser.CurrentChannelId, Body: chatUser.Name, UserCount: UserCount, CreatedDate: time.Now()}
	chatHistory := GetChatHistory(chatUser.CurrentChannelId)
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
	SendToOtherOnChannel(chatUser.Name+" has joined the channel.", chatUser, EventNotification)
	return nil
}

// HandleTypingEvent -
func HandleTypingEvent(body string, user *User) error {
	return nil
}
