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

func handleJoinEvent(body string, connection *websocket.Conn) error {
	return nil
}

func handleTypingEvent(body string, connection *websocket.Conn) error {
	return nil
}

func handleNameChangeEvent(body string, connection *websocket.Conn) error {
	return nil
}

func reader(connection *websocket.Conn) {
	for {
		var eventData util.EventData
		messageType, message, err := connection.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}
		if messageType == websocket.TextMessage {
			err := json.Unmarshal(message, &eventData)
			if err != nil {
				fmt.Println(err)
				return
			}
			var eventError error
			switch eventData.Event {
			case util.EventTypeJoin:
				eventError = handleJoinEvent(eventData.Body, connection)
			case util.EventTyping:
				eventError = handleTypingEvent(eventData.Body, connection)
			case util.EventMessage:
				eventError = handleMessageEvent(eventData.Body, connection)
			case util.EventTypeNameChange:
				eventError = handleNameChangeEvent(eventData.Body, connection)
			}
			if eventError != nil {
				fmt.Println(err)
			}
		}
		return
	}
}

func newChatConnection(connection *websocket.Conn) {
	fmt.Println("chatRequest(): Connection opened.")
	nano := strconv.Itoa(int(time.Now().UnixNano()))
	users = append(users, util.User{Name: "Anon" + nano, Connection: connection})
	reader(connection)
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
