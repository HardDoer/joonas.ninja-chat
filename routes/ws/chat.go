package ws;

import (
	"net/http"
	"fmt"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

func reader(connection *websocket.Conn) {
    for {
        messageType, message, err := connection.ReadMessage();
        if (err != nil) {
            fmt.Println(err);
            return
        }
        fmt.Println(string(message));

        if err := connection.WriteMessage(messageType, message); err != nil {
            fmt.Println(err);
            return
        }

    }
}

// ChatRequest - Handles chat requests via websocket.
func ChatRequest(responseWriter http.ResponseWriter, request *http.Request){
    wsConnection, err := upgrader.Upgrade(responseWriter, request, nil);
    if (err != nil) {
        fmt.Println(err);
	}
	reader(wsConnection);
}