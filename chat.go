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
	Event string `json:"event"`
	Body  string `json:"body"`
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

func handleMessageEvent(body string, connection *websocket.Conn) error {
	var senderName = ""
	for i := 0; i < len(users); i++ {
		if connection == users[i].Connection {
			senderName = users[i].Name
			break
		}
	}
	for i := 0; i < len(users); i++ {
		fmt.Println("SENDING TO: " + users[i].Name)
		response := eventData{Event: EventMessage, Body: senderName + ": " + body}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return err
		}
		if err := users[i].Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
			return err
		}
	}
	return nil
}

func handleJoin(chatUser *User, connection *websocket.Conn) error {
	var requestUser *User = chatUser
	fmt.Println("SENDING TO: " + requestUser.Name)
	response := eventData{Event: EventJoin, Body: requestUser.Name}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return err
	}
	if err := requestUser.Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
		return err
	}
	for i := 0; i < len(users); i++ {
		if users[i].Connection != connection {
			fmt.Println("SENDING TO: " + users[i].Name)
			response := eventData{Event: EventMessage, Body: requestUser.Name + " has joined the chat."}
			jsonResponse, err := json.Marshal(response)
			if err != nil {
				return err
			}
			if err := users[i].Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
				return err
			}
		}
	}
	return nil
}

func sendToAll(body string) {
	for i := 0; i < len(users); i++ {
		fmt.Println("SENDING TO: " + users[i].Name)
		response := eventData{Event: EventMessage, Body: body}
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

func getUserName(connection *websocket.Conn) string {
	for i := 0; i < len(users); i++ {
		if users[i].Connection == connection {
			return users[i].Name
		}
	}
	return ""
}

func handleTypingEvent(body string, connection *websocket.Conn) error {
	return nil
}

func handleNameChangeEvent(body string, connection *websocket.Conn) error {
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
			sendToAll(name + " has left the chatroom.")
			fmt.Println(connectionError)
		}
	}
	return
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
