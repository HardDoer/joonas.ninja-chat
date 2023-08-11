package main

import (
	"encoding/json"
	"log"
)

type chatHistory struct {
	Body      []EventData `json:"history"`
	UserCount int32       `json:"userCount"`
	Event     string      `json:"event"`
}

// updateChatHistory - Adds the parameter defined chat history entry to chat history
func updateChatHistory(jsonResponse []byte) {
	apiRequest("POST", apiRequestOptions{payload: jsonResponse}, "CHAT_HISTORY_URL", nil, nil)
}

func getChatHistory(channelId string) []byte {
	res, err := apiRequest("GET", apiRequestOptions{queryString: "?channelId=" + channelId}, "CHAT_HISTORY_URL", nil, nil)
	if err != nil {
		log.Print("getChatHistory():", err)
		return nil
	}
	var historyArray []EventData
	err = json.Unmarshal(res, &historyArray)
	if err != nil {
		log.Print("getChatHistory():", err)
		return nil
	}
	response := chatHistory{Event: EventChatHistory, Body: historyArray, UserCount: UserCount}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("getChatHistory():", err)
		return nil
	}
	return jsonResponse
}
