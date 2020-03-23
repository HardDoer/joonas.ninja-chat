package ws

import (
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

func reader(connection *websocket.Conn) {
	for {
		messageType, message, err := connection.ReadMessage()
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(message))

		// This works nicely. We can find the existing user connections in memory.
		for i := 0; i < len(users); i++ {
			fmt.Println(users[i].Name)
		}

		if err := connection.WriteMessage(messageType, message); err != nil {
			fmt.Println(err)
			return
		}
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
