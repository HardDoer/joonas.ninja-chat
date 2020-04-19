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
func sendToAll(body string, eventType string) {
	fmt.Println("sendToAll(): " + body)
	for i := 0; i < len(users); i++ {
		response := eventData{Event: eventType, Body: body, UserCount: len(users)}
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
	var connectionError error
	nano := strconv.Itoa(int(time.Now().UnixNano()))
	User := User{Name: "Anon" + nano, Connection: connection}
	users = append(users, User)
	err := handleJoin(&User, connection)
	if err != nil {
		connection.Close()
		removeUser(connection)
		fmt.Println(err)
	} else {
		connectionError = reader(connection)
		if connectionError != nil {
			connection.Close()
			name := getUserName(connection)
			removeUser(connection)
			sendToAll(name+" has left the chatroom.", EventNotification)
			fmt.Println(connectionError)
		}
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
	sendToAll(senderName+": "+body, EventMessage)
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

func reader(connection *websocket.Conn) error {
	for {
		var eventData eventData
		messageType, message, err := connection.ReadMessage()
		if err != nil {
			return err
		}
		if messageType == websocket.TextMessage {
			err := json.Unmarshal(message, &eventData)
			if err != nil {
				return err
			}
			var eventError error
			switch eventData.Event {
			case EventTyping:
				eventError = handleTypingEvent(eventData.Body, connection)
			case EventMessage:
				eventError = handleMessageEvent(eventData.Body, connection)
			case EventNameChange:
				eventError = handleNameChangeEvent(eventData.Body, connection)
			}
			if eventError != nil {
				return eventError
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
