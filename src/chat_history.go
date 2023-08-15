package main

import (
	"encoding/json"
	"errors"
	"log"
)

func newChatHistory(response any) (any, error) {
	// TODO. Tsekkaa toi tyyppi???
	parsed, ok := response.([]EventData)
	if (!ok) {
		return nil, errors.New("cannot parse response as EventData")
	}
	return chatHistory{Event: EventChatHistory, Body: parsed, UserCount: UserCount}, nil
}

type chatHistory struct {
	Body      []EventData `json:"history"`
	UserCount int32       `json:"userCount"`
	Event     string      `json:"event"`
}

// updateChatHistory - Adds the parameter defined chat history entry to chat history
func updateChatHistory(jsonResponse []byte) {
	apiRequest("POST", apiRequestOptions{payload: jsonResponse}, "CHAT_HISTORY_URL", nil, nil)
}

func getChatHistory(channelId string) any {
	res, err := apiRequest("GET", apiRequestOptions{queryString: "?channelId=" + channelId}, "CHAT_HISTORY_URL", nil, nil)
	if err != nil {
		log.Print("getChatHistory():", err)
		return nil
	}
	var eventData []EventData
	if err := json.Unmarshal(res, &eventData); err != nil {
		log.Print("getChatHistory():", err)
		return nil
	}
	return chatHistory{Event: EventChatHistory, Body: eventData, UserCount: UserCount}
}
