package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func chatHistoryTest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "{}")
}

func setupChatHistory() {
	http.HandleFunc("/test/chatHistory", chatHistoryTest)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Panic(err)
	}
}

func TestMessageJoinAndNormalMessage(t *testing.T) {
	var responseData EventData
	os.Setenv("CHAT_HISTORY_URL", "http://localhost:8080/test/chatHistory")
	go setupChatHistory()
	server := httptest.NewServer(http.HandlerFunc(ChatRequest))
	defer server.Close()
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.Nil(t, err)
	_, message, err := ws.ReadMessage()
	readerError := json.Unmarshal(message, &responseData)
	assert.Nil(t, readerError)
	assert.Equal(t, true, strings.HasPrefix(responseData.Body, "Anon") && responseData.Body != "Anon", "Name should be of the form AnonSomething")
	assert.Nil(t, err)
	testRequest := EventData{Event: EventMessage, Body: "Testing message"}
	jsonResponse, err := json.Marshal(testRequest)
	assert.Nil(t, err)
	assert.Nil(t, ws.WriteMessage(websocket.TextMessage, jsonResponse))
	_, message, err = ws.ReadMessage()
	readerError = json.Unmarshal(message, &responseData)
	assert.Nil(t, readerError)
	assert.Equal(t, true, responseData.Body == "Testing message" && responseData.UserCount == 1, responseData.Event == EventMessage)
}
