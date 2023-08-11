package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// EventData - A data structure that contains information about the current chat event.
type EventData struct {
	ChannelId   string    `json:"channelId"`
	Event       string    `json:"event"`
	Body        string    `json:"body"`
	UserCount   int32     `json:"userCount"`
	Name        string    `json:"name"`
	CreatedDate time.Time `json:"createdDate"`
}

// Users - A map containing all the connected users.
var Users sync.Map

// UserCount - Total count of connected users.
var UserCount int32

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func getUserName(connection *websocket.Conn) string {
	value, _ := Users.Load(connection)
	user := value.(*User)
	return user.Name
}

func removeUser(user *User) {
	Users.Delete(user)
	atomic.AddInt32(&UserCount, -1)
}

func ReplyMustBeLoggedIn(user *User) {
	SendToOne("Must be logged in for that command to work.", user, EventErrorNotification)
}

func NotEnoughParameters(user *User) {
	SendToOne("Not enough parameters. See '/help'", user, EventErrorNotification)
}

// SendToAll - sends the body string data to all connected clients
func SendToAll(body string, channelId string, name string, eventType string, restrictToChannel bool) {
	log.Println("sendToAll(): " + body)
	response := EventData{Event: eventType, ChannelId: channelId, Body: body, UserCount: UserCount, Name: name, CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("sendToAll():", err)
	}
	if eventType == EventMessage {
		UpdateChatHistory(jsonResponse)
	}
	Users.Range(func(key, value interface{}) bool {
		var userValue = value.(*User)
		if restrictToChannel {
			if userValue.CurrentChannelId == channelId {
				if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
					log.Print("sendToAll():", err)
				}
			}
		} else {
			if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
				log.Print("sendToAll():", err)
			}
		}
		return true
	})
}

// SendToOne - sends the body string data to a parameter defined client
func SendToOne(body string, user *User, eventType string) {
	log.Println("sendToOne(): " + body)
	response := EventData{Event: eventType, Body: body,
		UserCount: UserCount, Name: "", CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("sendToOne():", err)
	}
	if err := user.write(websocket.TextMessage, jsonResponse); err != nil {
		log.Print("sendToOne():", err)
	}
}

// SendToOtherOnChannel - sends the body string data to all connected clients on the same channel except the parameter given client
func SendToOtherOnChannel(body string, user *User, eventType string) {
	log.Print("SendToOtherOnChannel():", body)
	response := EventData{Event: eventType, ChannelId: user.CurrentChannelId, Body: body, UserCount: UserCount, CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("SendToOtherOnChannel():", err)
	}
	if eventType == EventMessage {
		UpdateChatHistory(jsonResponse)
	}
	Users.Range(func(key, value interface{}) bool {
		userValue := value.(*User)
		if userValue != user && userValue.CurrentChannelId == user.CurrentChannelId {
			if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
				log.Print("SendToOtherOnChannel():", err)
			}
		}
		return true
	})
}

// SendToOtherEverywhere - sends the body string data to all connected clients except the parameter given client
func SendToOtherEverywhere(body string, user *User, eventType string) {
	log.Print("SendToOtherEverywhere():", body)
	response := EventData{Event: eventType, ChannelId: user.CurrentChannelId, Body: body, UserCount: UserCount, CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("SendToOtherEverywhere():", err)
	}
	if eventType == EventMessage {
		UpdateChatHistory(jsonResponse)
	}
	Users.Range(func(key, value interface{}) bool {
		userValue := value.(*User)
		if userValue != user {
			if err := userValue.write(websocket.TextMessage, jsonResponse); err != nil {
				log.Print("SendToOtherEverywhere():", err)
			}
		}
		return true
	})
}

func newChatConnection(connection *websocket.Conn, cookie string) {
	log.Print("newChatConnection():", "Connection opened.")
	var validationRes tokenValidationRes
	var err error
	var newUser User
	var isClosed = false

	if cookie != "" {
		Users.Range(func(key, value interface{}) bool {
			var userValue = value.(*User)
			if len(userValue.Token) > 0 && userValue.Token == cookie {
				connection.Close()
				isClosed = true
				return false
			}
			return true
		})
		if isClosed == true {
			log.Print("newChatConnection(): Token already in use. Connection closed.")
			return
		}
		validationRes, err = validateToken(cookie)
		if err != nil {
			connection.Close()
			log.Print("newChatConnection():", err)
			return
		}
	}
	nano := strconv.Itoa(int(time.Now().UnixNano()))
	if len(validationRes.Username) > 0 {
		newUser = User{Name: validationRes.Username, CurrentChannelId: validationRes.DefaultChannel, Connection: connection, Token: cookie}
	} else {
		newUser = User{Name: "Anon" + nano, Connection: connection, Token: cookie}
	}
	Users.Store(&newUser, &newUser)
	atomic.AddInt32(&UserCount, 1)
	SendToOtherEverywhere(newUser.Name+" has connected.", &newUser, EventNotification)
	err = HandleJoin(&newUser)
	if err != nil {
		connection.Close()
		removeUser(&newUser)
		log.Print("newChatConnection():", err)
	} else {
		if len(newUser.Token) > 0 {
			SendToOne("Logged in successfully.", &newUser, EventLogin)
		}
		go reader(&newUser)
		go heartbeat(&newUser)
	}
}

func reader(user *User) {
	var readerError error
	defer func() {
		log.Print("reader():", readerError)
		user.Connection.Close()
		key, _ := Users.Load(user)
		user := key.(*User)
		removeUser(user)
		SendToAll(user.Name+" has disconnected.", user.CurrentChannelId, "", EventNotification, false)
	}()
	user.Connection.SetReadLimit(maxMessageSize)
	user.Connection.SetReadDeadline(time.Now().Add(pongWait))
	user.Connection.SetPongHandler(func(string) error { user.Connection.SetReadDeadline(time.Now().Add(pongWait)); return nil })
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
				readerError = HandleTypingEvent(EventData.Body, user)
			case EventMessage:
				readerError = HandleMessageEvent(EventData.Body, user)
			}
			if readerError != nil {
				return
			}
		}
	}
}

// ChatRequest - A chat request.
func ChatRequest(responseWriter http.ResponseWriter, request *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		allowedOrigin, found := os.LookupEnv("ALLOWED_ORIGIN")
		if found {
			return r.Header.Get("Origin") == "http://"+allowedOrigin ||
				r.Header.Get("Origin") == "https://"+allowedOrigin ||
				r.Header.Get("Origin") == "https://www."+allowedOrigin ||
				r.Header.Get("Origin") == "http://www."+allowedOrigin
		}
		return true
	}
	wsConnection, err := upgrader.Upgrade(responseWriter, request, nil)
	if err != nil {
		log.Print("ChatRequest():", err)
	} else {
		newChatConnection(wsConnection, request.Header.Get("Cookie"))
	}
}
