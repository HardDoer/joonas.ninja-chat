package ws

import (
	"container/list"
	"fmt"
	"net/http"
	"strings"

	"joonas.ninja-chat/util"

	"github.com/gorilla/websocket"
)

type chatRequest struct {
	Type    string
	Message string
}

var users = list.New()

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

		if err := connection.WriteMessage(messageType, message); err != nil {
			fmt.Println(err)
			return
		}
	}
}

func newChatConnection(connection *websocket.Conn) {
	fmt.Println("chatRequest(): Connection opened.")
	users.PushBack(util.User{Name: "kikkare", Connection: connection});
	reader(connection)
}

// WebsocketRequest - Handles websocket requests and conveys them to the handler depending on request path.
func WebsocketRequest(responseWriter http.ResponseWriter, request *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	wsConnection, err := upgrader.Upgrade(responseWriter, request, nil)
	if err != nil {
		fmt.Println(err)
	}
	pathArray := strings.Split(request.RequestURI, "/")

	if pathArray[len(pathArray)-1] == "chat" {
		newChatConnection(wsConnection)
	}
}
