package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// EventData - A data structure that contains information about the current chat event.
type EventData struct {
	Event       string    `json:"event"`
	Body        string    `json:"body"`
	UserCount   int32     `json:"userCount"`
	Name        string    `json:"name"`
	CreatedDate time.Time `json:"createdDate"`
	Auth        string    `json:"auth"`
}

type chatHistory struct {
	Body      []EventData `json:"history"`
	UserCount int32       `json:"userCount"`
	Event     string      `json:"event"`
}

// Users - A map containing all the connected users.
var Users sync.Map
var userCount int32

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func updateChatHistory(jsonResponse []byte) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", os.Getenv("CHAT_HISTORY_URL"), bytes.NewBuffer(jsonResponse))
	if err != nil {
		log.Print("updateChatHistory():", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", `Basic `+
		base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("APP_KEY"))))
	historyResponse, err := client.Do(req)
	if historyResponse != nil && historyResponse.Status != "200 OK" {
		log.Print("updateChatHistory():", "Error response "+historyResponse.Status)
	}
	if err != nil {
		log.Print("updateChatHistory():", err)
	}
	defer historyResponse.Body.Close()
}

func getChatHistory() []byte {
	client := &http.Client{}
	req, err := http.NewRequest("GET", os.Getenv("CHAT_HISTORY_URL"), nil)
	req.Header.Add("Authorization", `Basic `+
		base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("APP_KEY"))))
	historyResponse, err := client.Do(req)
	if historyResponse != nil && historyResponse.Status != "200 OK" {
		log.Print("getChatHistory():", "Error response "+historyResponse.Status)
		return nil
	}
	if err != nil {
		log.Print("getChatHistory():", err)
		return nil
	}
	defer historyResponse.Body.Close()
	body, err := ioutil.ReadAll(historyResponse.Body)
	if err != nil {
		log.Print("getChatHistory():", err)
		return nil
	}
	var historyArray []EventData
	err = json.Unmarshal(body, &historyArray)
	if err != nil {
		log.Print("getChatHistory():", err)
		return nil
	}
	response := chatHistory{Event: EventChatHistory, Body: historyArray, UserCount: userCount}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("getChatHistory():", err)
		return nil
	}
	return jsonResponse
}

func removeUser(user *User) {
	Users.Delete(user)
	atomic.AddInt32(&userCount, -1)
}

// sendToAll - sends the body string data to all connected clients
func sendToAll(body string, name string, eventType string) {
	log.Println("sendToAll(): " + body)
	response := EventData{Event: eventType, Body: body, UserCount: userCount, Name: name, CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("sendToAll():", err)
	}
	if eventType == EventMessage {
		updateChatHistory(jsonResponse)
	}
	Users.Range(func(key, value interface{}) bool {
		var userValue = value.(*User)
		if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
			log.Print("sendToAll():", err)
		}
		return true
	})
}

// SendToOne - sends the body string data to a parameter defined client
func SendToOne(body string, user *User, eventType string) {
	log.Println("sendToOne(): " + body)
	response := EventData{Event: eventType, Body: body,
		UserCount: userCount, Name: "", CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("sendToOne():", err)
	}
	if err := user.write(websocket.TextMessage, jsonResponse); err != nil {
		log.Print("sendToOne():", err)
	}
}

// sendToOther - sends the body string data to all connected clients except the parameter given client
func sendToOther(body string, user *User, eventType string) {
	log.Print("sendToOther():", body)
	response := EventData{Event: eventType, Body: body, UserCount: userCount, CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("sendToOther():", err)
	}
	if eventType == EventMessage {
		updateChatHistory(jsonResponse)
	}
	Users.Range(func(key, value interface{}) bool {
		userValue := value.(*User)
		if userValue != user {
			if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
				log.Print("sendToOther():", err)
			}
		}
		return true
	})
}

func getUserName(connection *websocket.Conn) string {
	value, _ := Users.Load(connection)
	user := value.(*User)
	return user.Name
}

func newChatConnection(connection *websocket.Conn) {
	log.Print("chatRequest():", "Connection opened.")
	nano := strconv.Itoa(int(time.Now().UnixNano()))
	newUser := User{Name: "Anon" + nano, Connection: connection}
	Users.Store(&newUser, &newUser)
	atomic.AddInt32(&userCount, 1)
	err := handleJoin(&newUser)
	if err != nil {
		connection.Close()
		removeUser(&newUser)
		log.Print("newChatConnection():", err)
	} else {
		go reader(&newUser)
		if userCount == 1 {
			go heartbeat()
		}
	}
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

func handleMessageEvent(body string, user *User) error {
	var senderName = ""
	if len(body) < 512 {
		if strings.Index(body, "/") != 0 {
			value, _ := Users.Load(user)
			user := value.(*User)
			senderName = user.Name
			sendToAll(body, senderName, EventMessage)
		} else {
			handleCommand(body, user)
		}
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä viestistä.
		log.Println("Message is too long")
	}
	return nil
}

func handleJoin(chatUser *User) error {
	response := EventData{Event: EventJoin, Body: chatUser.Name, UserCount: userCount, CreatedDate: time.Now()}
	chatHistory := getChatHistory()
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
	sendToOther(chatUser.Name+" has joined the chat.", chatUser, EventNotification)
	return nil
}

func handleTypingEvent(body string, user *User) error {
	return nil
}

func handleNameChangeEvent(body string, user *User) error {
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
		sendToOther(originalName+" is now called "+body, user, EventNotification)
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä nimestä. Lisää vaikka joku error-tyyppi.
		log.Println("New name is too long or too short")
	}
	return nil
}

func reader(user *User) {
	var readerError error
	defer func() {
		log.Print("reader():", readerError)
		user.Connection.Close()
		key, _ := Users.Load(user)
		user := key.(*User)
		removeUser(user)
		sendToAll(user.Name+" has left the chatroom.", "", EventNotification)
	}()
	for {
		var EventData EventData
		messageType, message, readerError := user.Connection.ReadMessage()
		if readerError != nil {
			return
		}
		if messageType == websocket.TextMessage {
			readerError = json.Unmarshal(message, &EventData)
			if readerError != nil {
				return
			}
			switch EventData.Event {
			case EventTyping:
				readerError = handleTypingEvent(EventData.Body, user)
			case EventMessage:
				readerError = handleMessageEvent(EventData.Body, user)
			case EventNameChange:
				readerError = handleNameChangeEvent(EventData.Body, user)
			}
			if readerError != nil {
				return
			}
		}
	}
}

func heartbeat() {
	for {
		if userCount == 0 {
			return
		}
		time.Sleep(2 * time.Second)
		log.Println("PING")
		Users.Range(func(key, value interface{}) bool {
			userValue := value.(*User)
			if err := userValue.write(websocket.PingMessage, nil); err != nil {
				log.Print("heartbeat():", err)
			}
			return true
		})
	}
}

// ChatRequest - A chat request.
func ChatRequest(responseWriter http.ResponseWriter, request *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		allowedOrigin, found := os.LookupEnv("ALLOWED_ORIGIN")
		if found {
			return r.Header.Get("Origin") == "http://"+allowedOrigin ||
				r.Header.Get("Origin") == "https://"+allowedOrigin
		}
		return true
	}
	wsConnection, err := upgrader.Upgrade(responseWriter, request, nil)
	if err != nil {
		log.Print("ChatRequest():", err)
	} else {
		newChatConnection(wsConnection)
	}
}
