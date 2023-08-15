package main

import (
	"encoding/json"
	"errors"
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

type messageFn func (user *User, jsonResponse []byte) func(key any, value any) bool

func getEvent(event string) (func(string, *User), bool) {
	var events = map[string]func(string, *User){
		EventTyping: handleTypingEvent,
		EventMessage: handleMessageEvent,
	}
	eventFn, ok := events[event]
	return eventFn, ok
}

// Users - A map containing all the connected users.
var Users sync.Map

// UserCount - Total count of connected users.
var UserCount int32

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func removeUser(user *User) {
	Users.Delete(user)
	atomic.AddInt32(&UserCount, -1)
}

func replyMustBeLoggedIn() error {
	return errors.New("must be logged in for that command to work")
}

func notEnoughParameters() error {
	return errors.New("not enough parameters. See '/help'")
}

func sendToOtherEverywhere(body string, user *User, eventType string, displayName bool, updateHistory bool) {
	sendMultipleMessages(user, body, eventType, displayName, updateHistory, sendToOtherEverywhereFilter)
}

func sendToOtherOnChannel(body string, user *User, eventType string, displayName bool, updateHistory bool) {
	sendMultipleMessages(user, body, eventType, displayName, updateHistory, sendToOtherOnChannelFilter)
}

func sendToAllOnChannel(body string, user *User, eventType string, displayName bool, updateHistory bool) {
	sendMultipleMessages(user, body, eventType, displayName, updateHistory, sendToAllOnChannelFilter)
}

func sendToAll(body string, user *User, eventType string, displayName bool, updateHistory bool) {
	sendMultipleMessages(user, body, eventType, displayName, updateHistory, sendToAllFilter)
}

func marshalAndWriteToStream(user *User, response any) {
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("marshalAndWriteToStream():", err)
	}
	if err := user.write(websocket.TextMessage, jsonResponse); err != nil {
		log.Print("marshalAndWriteToStream():", err)
	}
}

// sends the body string data to a parameter defined client
func sendSystemMessage(body string, user *User, eventType string) {
	log.Println("sendOneMessage(): " + body)
	response := EventData{Event: eventType, Body: body,
		UserCount: UserCount, Name: SystemName, CreatedDate: time.Now()}
	marshalAndWriteToStream(user, response)
}

// send multiple messages using the provided filterFunction
func sendMultipleMessages(user *User, body string, eventType string, displayName bool, updateHistory bool, filterFn messageFn) {
	log.Print("sendMultipleMessages():", body)
	var response EventData
	var name string
	if (!displayName) {
		name = SystemName
	} else {
		name = user.Name
	}
	response = EventData{Event: eventType, ChannelId: user.CurrentChannelId, Body: body, Name: name, UserCount: UserCount, CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("sendMultipleMessages():", err)
	}
	if updateHistory {
		updateChatHistory(jsonResponse)
	}
	fn := filterFn(user, jsonResponse)
	Users.Range(fn)
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
		if isClosed {
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
	sendToOtherEverywhere(newUser.Name+" has connected.", &newUser, EventNotification, false, false)
	handleJoin(&newUser)
	if len(newUser.Token) > 0 {
		sendSystemMessage("Logged in successfully.", &newUser, EventLogin)
	}
	go reader(&newUser)
	go heartbeat(&newUser)
}

func reader(user *User) {
	var readerError error
	defer func() {
		log.Print("reader():", readerError)
		user.Connection.Close()
		key, _ := Users.Load(user)
		user := key.(*User)
		removeUser(user)
		sendToAll(user.Name+" has disconnected.", user, EventNotification, false, false)
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
			eventFn, ok := getEvent(EventData.Event)
			if readerError != nil || !ok {
				log.Println("event not recognized")
				return
			}
			eventFn(EventData.Body, user)
		}
	}
}

// ChatRequest - A chat request.
func chatRequest(responseWriter http.ResponseWriter, request *http.Request) {
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
