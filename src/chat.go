package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// EventData - A data structure that contains information about the current chat event.
type eventData struct {
	Event       string    `json:"event"`
	Body        string    `json:"body"`
	UserCount   int       `json:"userCount"`
	Name        string    `json:"name"`
	CreatedDate time.Time `json:"CreatedDate"`
}

var users []User

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func updateChatHistory(jsonResponse []byte) {
	historyResponse, err := http.Post(os.Getenv("CHAT_HISTORY_URL"), "application/json", bytes.NewBuffer(jsonResponse))
	if err != nil && historyResponse.Status != "200" {
		log.Printf("updateChatHistory(): ")
		log.Println(err)
	}
}

func removeUser(connection *websocket.Conn) {
	var newUsers []User
	for i := 0; i < len(users); i++ {
		if connection != users[i].Connection {
			newUsers = append(newUsers, users[i])
		}
	}
	users = newUsers
}

// sendToAll - sends the body string data to all connected clients
func sendToAll(body string, name string, eventType string) {
	log.Println("sendToAll(): " + body)
	response := eventData{Event: eventType, Body: body, UserCount: len(users), Name: name, CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Printf("sendToAll(): ")
		log.Println(err)
	}
	updateChatHistory(jsonResponse)
	for i := 0; i < len(users); i++ {
		if err := users[i].Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
			log.Printf("sendToAll(): ")
			log.Println(err)
		}
	}
}

// sendToOther - sends the body string data to all connected clients except the parameter given client
func sendToOther(body string, connection *websocket.Conn, eventType string) {
	log.Println("sendToOther(): " + body)
	response := eventData{Event: eventType, Body: body, UserCount: len(users), CreatedDate: time.Now()}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Printf("sendToOther(): ")
		log.Println(err)
	}
	updateChatHistory(jsonResponse)
	for i := 0; i < len(users); i++ {
		if users[i].Connection != connection {
			if err := users[i].Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
				log.Printf("sendToOther(): ")
				log.Println(err)
			}
		}
	}
}

func getUserName(connection *websocket.Conn) string {
	for i := 0; i < len(users); i++ {
		if users[i].Connection == connection {
			return users[i].Name
		}
	}
	return ""
}

func newChatConnection(connection *websocket.Conn) {
	log.Println("chatRequest(): Connection opened.")
	nano := strconv.Itoa(int(time.Now().UnixNano()))
	User := User{Name: "Anon" + nano, Connection: connection}
	users = append(users, User)
	err := handleJoin(&User, connection)
	if err != nil {
		connection.Close()
		removeUser(connection)
		log.Println(err)
	} else {
		go reader(connection)
		go writer(connection)
	}
	return
}

func handleMessageEvent(body string, connection *websocket.Conn) error {
	var senderName = ""
	for i := 0; i < len(users); i++ {
		if connection == users[i].Connection {
			senderName = users[i].Name
			break
		}
	}
	sendToAll(body, senderName, EventMessage)
	return nil
}

func handleJoin(chatUser *User, connection *websocket.Conn) error {
	var requestUser *User = chatUser
	response := eventData{Event: EventJoin, Body: requestUser.Name, UserCount: len(users), CreatedDate: time.Now()}
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
		for i := 0; i < len(users); i++ {
			log.Println("handleNameChangeEvent(): User " + users[i].Name + " is changing name.")
			if connection == users[i].Connection {
				originalName = users[i].Name
				users[i].Name = body
				response := eventData{Event: EventNameChange, Body: users[i].Name, UserCount: len(users), CreatedDate: time.Now()}
				jsonResponse, err := json.Marshal(response)
				if err != nil {
					return err
				}
				if err := users[i].Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
					return err
				}
				break
			}
		}
		sendToOther(originalName+" is now called "+body, connection, EventNotification)
	} else {
		// TODO. Palauta joku virhe käyttäjälle liian pitkästä nimestä. Lisää vaikka joku error-tyyppi.
		log.Println("New name is too long")
	}
	return nil
}

func writer(connection *websocket.Conn) {
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

func reader(connection *websocket.Conn) {
	var readerError error
	defer func() {
		log.Println(readerError)
		connection.Close()
		name := getUserName(connection)
		removeUser(connection)
		sendToAll(name+" has left the chatroom.", "", EventNotification)
	}()
	for {
		var eventData eventData
		messageType, message, readerError := connection.ReadMessage()
		if readerError != nil {
			return
		}
		if messageType == websocket.TextMessage {
			readerError = json.Unmarshal(message, &eventData)
			if readerError != nil {
				return
			}
			switch eventData.Event {
			case EventTyping:
				readerError = handleTypingEvent(eventData.Body, connection)
			case EventMessage:
				readerError = handleMessageEvent(eventData.Body, connection)
			case EventNameChange:
				readerError = handleNameChangeEvent(eventData.Body, connection)
			}
			if readerError != nil {
				return
			}
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
