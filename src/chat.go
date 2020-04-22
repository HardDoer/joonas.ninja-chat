package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// EventData - A data structure that contains information about the current chat event.
type eventData struct {
	Event     string `json:"event"`
	Body      string `json:"body"`
	UserCount int    `json:"userCount"`
	Name      string `json:"name"`
}

var users []User

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
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
	fmt.Println("sendToAll(): " + body)
	for i := 0; i < len(users); i++ {
		response := eventData{Event: eventType, Body: body, UserCount: len(users), Name: name}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			fmt.Printf("sendToAll(): ")
			fmt.Println(err)
		}
		if err := users[i].Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
			fmt.Printf("sendToAll(): ")
			fmt.Println(err)
		}
	}
}

// sendToOther - sends the body string data to all connected clients except the parameter given client
func sendToOther(body string, connection *websocket.Conn, eventType string) {
	fmt.Println("sendToOther(): " + body)
	for i := 0; i < len(users); i++ {
		if users[i].Connection != connection {
			response := eventData{Event: eventType, Body: body, UserCount: len(users)}
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				fmt.Printf("sendToOther(): ")
				fmt.Println(err)
			}
			if err := users[i].Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
				fmt.Printf("sendToOther(): ")
				fmt.Println(err)
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
	fmt.Println("chatRequest(): Connection opened.")
	nano := strconv.Itoa(int(time.Now().UnixNano()))
	User := User{Name: "Anon" + nano, Connection: connection}
	users = append(users, User)
	err := handleJoin(&User, connection)
	if err != nil {
		connection.Close()
		removeUser(connection)
		fmt.Println(err)
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
	response := eventData{Event: EventJoin, Body: requestUser.Name, UserCount: len(users)}
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
			fmt.Println("handleNameChangeEvent(): User " + users[i].Name + " is changing name.")
			if connection == users[i].Connection {
				originalName = users[i].Name
				users[i].Name = body
				response := eventData{Event: EventNameChange, Body: users[i].Name, UserCount: len(users)}
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
		fmt.Println("New name is too long")
	}
	return nil
}

func writer(connection *websocket.Conn) {
	defer func() {
		connection.Close()
	}()
	for {
		time.Sleep(2 * time.Second)
		fmt.Println("PING");
		if err := connection.WriteMessage(websocket.PingMessage, nil); err != nil {
			fmt.Println(err)
			return
		}
	}
}

func reader(connection *websocket.Conn) {
	var readerError error
	defer func() {
		fmt.Println(readerError)
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
		fmt.Println(err)
	}
	newChatConnection(wsConnection)
}
