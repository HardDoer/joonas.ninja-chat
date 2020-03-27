package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"joonas.ninja-chat/util"

	"github.com/gorilla/websocket"
)

var users []util.User

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func removeUser(connection *websocket.Conn) {
	var newUsers []util.User
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
		response := util.EventData{Event: util.EventMessage, Body: senderName + ": " + body}
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

func handleJoin(user *util.User, connection *websocket.Conn) error {
	// TODO: Announce to everyone that this user has joined the chat.
	var requestUser *util.User = user
	fmt.Println("SENDING TO: " + requestUser.Name)
	response := util.EventData{Event: util.EventJoin, Body: requestUser.Name}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return err
	}
	if err := requestUser.Connection.WriteMessage(websocket.TextMessage, jsonResponse); err != nil {
		return err
	}
	return nil
}

func handleTypingEvent(body string, connection *websocket.Conn) error {
	return nil
}

func handleNameChangeEvent(body string, connection *websocket.Conn) error {
	return nil
}

func reader(connection *websocket.Conn) error {
	for {
		var eventData util.EventData
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
			case util.EventTyping:
				eventError = handleTypingEvent(eventData.Body, connection)
			case util.EventMessage:
				eventError = handleMessageEvent(eventData.Body, connection)
			case util.EventNameChange:
				eventError = handleNameChangeEvent(eventData.Body, connection)
			}
			if eventError != nil {
				return eventError
			}
		}
		return nil
	}
}

func newChatConnection(connection *websocket.Conn) {
	fmt.Println("chatRequest(): Connection opened.")
	var connectionError error
	nano := strconv.Itoa(int(time.Now().UnixNano()))
	user := util.User{Name: "Anon" + nano, Connection: connection}
	users = append(users, user)
	err := handleJoin(&user, connection)
	if err != nil {
		connection.Close()
		removeUser(connection);
		fmt.Println(err)
	} else {
		connectionError = reader(connection)
		if connectionError != nil {
			connection.Close()
			removeUser(connection);
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
