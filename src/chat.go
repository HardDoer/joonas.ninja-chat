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
	UserCount   int32    `json:"userCount"`
	Name        string    `json:"name"`
	CreatedDate time.Time `json:"createdDate"`
	Auth        string    `json:"auth"`
}

type chatHistory struct {
	Body      []EventData `json:"history"`
	UserCount int32         `json:"userCount"`
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
		log.Printf("updateChatHistory(): ")
		log.Println(err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", `Basic `+
		base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("APP_KEY"))))
	historyResponse, err := client.Do(req)
	if historyResponse != nil && historyResponse.Status != "200 OK" {
		log.Printf("updateChatHistory(): Error response " + historyResponse.Status)
	}
	if err != nil {
		log.Printf("updateChatHistory():")
		log.Println(err)
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
		log.Printf("getChatHistory(): Error response " + historyResponse.Status)
		return nil
	}
	if err != nil {
		log.Printf("getChatHistory(): ")
		log.Println(err)
		return nil
	}
	defer historyResponse.Body.Close()
	body, err := ioutil.ReadAll(historyResponse.Body)
	if err != nil {
		log.Printf("getChatHistory(): ")
		log.Println(err)
		return nil
	}
	var historyArray []EventData
	err = json.Unmarshal(body, &historyArray)
	if err != nil {
		log.Printf("getChatHistory(): ")
		log.Println(err)
		return nil
	}
	response := chatHistory{Event: EventChatHistory, Body: historyArray, UserCount: userCount}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Printf("getChatHistory(): ")
		log.Println(err)
		return nil
	}
	return jsonResponse
}

func removeUser(connection *websocket.Conn) {
	Users.Delete(connection)
	atomic.AddInt32(&userCount, -1)
}

// sendToAll - sends the body string data to all connected clients
func sendToAll(body string, name string, eventType string) {
	log.Println("sendToAll(): " + body)
	response := EventData{Event: eventType, Body: body, UserCount: userCount, Name: name, CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Printf("sendToAll(): ")
		log.Println(err)
	}
	if eventType == EventMessage {
		updateChatHistory(jsonResponse)
	}
	Users.Range(func(key, value interface{}) bool {
		var connectionKey = key.(*websocket.Conn)
		if err := connectionKey.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
			log.Printf("sendToAll(): ")
			log.Println(err)
		}
		return true
	})
}

// SendToOne - sends the body string data to a parameter defined client
func SendToOne(body string, connection *websocket.Conn, eventType string) {
	log.Println("sendToOne(): " + body)
	response := EventData{Event: eventType, Body: body,
		UserCount: userCount, Name: "", CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Printf("sendToOne(): ")
		log.Println(err)
	}
	if err := connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
		log.Printf("sendToOne(): ")
		log.Println(err)
	}
}

// sendToOther - sends the body string data to all connected clients except the parameter given client
func sendToOther(body string, connection *websocket.Conn, eventType string) {
	log.Println("sendToOther(): " + body)
	response := EventData{Event: eventType, Body: body, UserCount: userCount, CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Printf("sendToOther(): ")
		log.Println(err)
	}
	if eventType == EventMessage {
		updateChatHistory(jsonResponse)
	}
	Users.Range(func(key, value interface {}) bool {
		connectionKey := key.(*websocket.Conn)
		if connectionKey != connection {
			if err := connectionKey.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
				log.Printf("sendToOther(): ")
				log.Println(err)
			}
		}
		return true
	})
}

func getUserName(connection *websocket.Conn) string {
	value, _:= Users.Load(connection)
	user := value.(User)
	return user.Name
}

func newChatConnection(connection *websocket.Conn) {
	log.Println("chatRequest(): Connection opened.")
	nano := strconv.Itoa(int(time.Now().UnixNano()))
	newUser := User{Name: "Anon" + nano, Connection: connection}
	Users.Store(connection, newUser)
	atomic.AddInt32(&userCount, 1)
	err := handleJoin(&newUser, connection)
	if err != nil {
		connection.Close()
		removeUser(connection)
		log.Println(err)
	} else {
		go reader(connection)
		go heartbeat(connection)
	}
	return
}

func handleCommand(body string, connection *websocket.Conn) {
	var splitBody = strings.Split(body, "/")
	splitBody = strings.Split(splitBody[1], " ")
	command := splitBody[0]
	switch command {
	case CommandWho:
		HandleWhoCommand(connection)
		/*
	case CommandChannel:
		HandleChannelCommand(splitBody, connection)
		*/
	default:
		SendToOne("Command "+"'"+body+"' not recognized.", connection, EventNotification)
	}
}

func handleMessageEvent(body string, connection *websocket.Conn) error {
	var senderName = ""
	if len(body) < 512 {
		if strings.Index(body, "/") != 0 {
			value, _ := Users.Load(connection)
			user := value.(User)
			senderName = user.Name
			sendToAll(body, senderName, EventMessage)
		} else {
			handleCommand(body, connection)
		}
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä viestistä.
		log.Println("Message is too long")
	}
	return nil
}

func handleJoin(chatUser *User, connection *websocket.Conn) error {
	var requestUser *User = chatUser
	response := EventData{Event: EventJoin, Body: requestUser.Name, UserCount: userCount, CreatedDate: time.Now()}
	chatHistory := getChatHistory()
	if chatHistory != nil {
		if err := requestUser.Connection.WriteMessage(websocket.TextMessage, chatHistory); err != nil {
			return err
		}
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return err
	}
	if err := requestUser.Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
		return err
	}
	sendToOther(requestUser.Name+" has joined the chat.", connection, EventNotification)
	return nil
}

func handleTypingEvent(body string, connection *websocket.Conn) error {
	return nil
}

func handleNameChangeEvent(body string, connection *websocket.Conn) error {
	if len(body) < 64 {
		var originalName string
		key, _ := Users.Load(connection)
		user := key.(User)
		log.Println("handleNameChangeEvent(): User " + user.Name + " is changing name.")
		originalName = user.Name
		user.Name = body
		Users.Store(connection, user)
		response := EventData{Event: EventNameChange, Body: user.Name, UserCount: userCount, CreatedDate: time.Now()}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return err
		}
		if err := connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
			return err
		}
		sendToOther(originalName+" is now called "+body, connection, EventNotification)
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä nimestä. Lisää vaikka joku error-tyyppi.
		log.Println("New name is too long")
	}
	return nil
}

func reader(connection *websocket.Conn) {
	var readerError error
	defer func() {
		log.Println(readerError)
		connection.Close()
		key, _ := Users.Load(connection)
		user := key.(User)
		removeUser(connection)
		sendToAll(user.Name+" has left the chatroom.", "", EventNotification)
	}()
	for {
		var EventData EventData
		messageType, message, readerError := connection.ReadMessage()
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
				readerError = handleTypingEvent(EventData.Body, connection)
			case EventMessage:
				readerError = handleMessageEvent(EventData.Body, connection)
			case EventNameChange:
				readerError = handleNameChangeEvent(EventData.Body, connection)
			}
			if readerError != nil {
				return
			}
		}
	}
}

func heartbeat(connection *websocket.Conn) {
	defer func() {
		connection.Close()
	}()
	for {
		time.Sleep(2 * time.Second)
		log.Println("PING")
		if err := connection.WriteMessage(websocket.PingMessage, nil); err != nil {
			log.Println(err)
			return
		}
	}
}

// ChatRequest - A chat request.
func ChatRequest(responseWriter http.ResponseWriter, request *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	wsConnection, err := upgrader.Upgrade(responseWriter, request, nil)
	if err != nil {
		log.Println(err)
	}
	newChatConnection(wsConnection)
}
