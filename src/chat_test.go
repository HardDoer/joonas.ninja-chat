package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TODO. Return something more useful.
func chatHistoryTest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "{}")
}

func testSetup(t *testing.T) (*websocket.Conn, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(ChatRequest))
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	assert.Nil(t, err)
	return ws, server
}

func setupChatHistory() *httptest.Server {
	chatHistoryServer := httptest.NewServer(http.HandlerFunc(chatHistoryTest))
	os.Setenv("CHAT_HISTORY_URL", chatHistoryServer.URL)
	return chatHistoryServer
}

func TestJoin(t *testing.T) {
	var responseData EventData
	ws, server := testSetup(t)
	defer func() {
		server.Close()
		ws.Close()
	}()
	_, message, err := ws.ReadMessage()
	assert.Nil(t, err)
	readerError := json.Unmarshal(message, &responseData)
	assert.Nil(t, readerError)
	assert.Equal(t, true, strings.HasPrefix(responseData.Body, "Anon") && responseData.Body != "Anon",
		"Name should be of the form AnonSomething")
}

func TestSendMessage(t *testing.T) {
	var responseData EventData
	ws, server := testSetup(t)
	chathistoryServer := setupChatHistory()
	defer func() {
		server.Close()
		ws.Close()
		chathistoryServer.Close()
	}()
	_, _, err := ws.ReadMessage()
	assert.Nil(t, err)
	testRequest := EventData{Event: EventMessage, Body: "Testing message"}
	jsonResponse, err := json.Marshal(testRequest)
	assert.Nil(t, err)
	assert.Nil(t, ws.WriteMessage(websocket.TextMessage, jsonResponse))
	_, message, err := ws.ReadMessage()
	assert.Nil(t, err)
	readerError := json.Unmarshal(message, &responseData)
	assert.Nil(t, readerError)
	assert.Equal(t, true, responseData.Body == "Testing message" && responseData.UserCount == 1, responseData.Event == EventMessage,
		"Response to a normal chatmessage should be structured as expected.")
}

func TestChangeName(t *testing.T) {
	var responseData EventData
	ws, server := testSetup(t)
	defer func() {
		server.Close()
		ws.Close()
	}()
	_, _, err := ws.ReadMessage()
	assert.Nil(t, err)
	testRequest := EventData{Event: EventNameChange, Body: "TestDude"}
	jsonResponse, err := json.Marshal(testRequest)
	assert.Nil(t, err)
	assert.Nil(t, ws.WriteMessage(websocket.TextMessage, jsonResponse))
	_, message, err := ws.ReadMessage()
	assert.Nil(t, err)
	readerError := json.Unmarshal(message, &responseData)
	assert.Nil(t, readerError)
	assert.Equal(t, true, responseData.Body == "TestDude" && responseData.UserCount == 1, responseData.Event == EventNameChange,
		"nameChange-event should return the user set name in the response and the response structure should be proper.")
}
