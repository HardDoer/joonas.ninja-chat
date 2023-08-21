package main

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
)

func getChannelCommand(command string) (func([]string, *User) error, bool) {
	var commands = map[string]func([]string, *User) error{
		"create":  handleChannelCreate,
		"invite":  handleChannelInvite,
		"join":    handleChannelJoin,
		"list":    handleChannelList,
		"default": handleChannelDefault,
	}
	commandFn, ok := commands[command]
	return commandFn, ok
}

func handleChannelCreate(commands []string, user *User) error {
	if len(commands) < 3 {
		return errors.New("no empty names")
	}
	var parameter1 = commands[2]
	var parameter2 = false
	if len(commands) >= 4 && commands[3] == "private" {
		parameter2 = true
	}
	parameter1 = strings.ReplaceAll(parameter1, " ", "")
	if parameter1 == "" {
		return errors.New("no empty names")
	}
	if parameter1 == PublicChannelName {
		return errors.New("that is a reserved name. Try a different name for your channel")
	}
	if len(parameter1) <= 16 {
		jsonResponse, _ := json.Marshal(channelDTO{Name: parameter1, CreatorToken: user.Token, Private: parameter2})
		_, err := apiRequest("POST", newApiRequestOptions(&apiRequestOptions{payload: jsonResponse}), "CHAT_CHANNEL_URL", func(response []byte) []byte {
			sendSystemMessage("Successfully created channel: '"+parameter1+"'. private: "+strconv.FormatBool(parameter2), user, EventNotification)
			return nil
		})
		if err != nil {
			return errors.New("error creating channel")
		}
	}
	return nil
}

func handleChannelInvite(commands []string, user *User) error {
	if len(commands) != 4 {
		return errors.New("insufficient parameters")
	}
	var parameter1 = commands[2]
	var parameter2 = commands[3]
	jsonResponse, _ := json.Marshal(channelInviteDTO{Name: parameter1, CreatorToken: user.Token, NewUser: parameter2})
	_, err := apiRequest("PUT", newApiRequestOptions(&apiRequestOptions{payload: jsonResponse}), "CHAT_CHANNEL_INVITE_URL", func(response []byte) []byte {
		sendSystemMessage("Invite sent successfully to: "+parameter2, user, EventNotification)
		return nil
	})
	if err != nil {
		return errors.New("error sending channel invite")
	}
	return nil
}

func handleChannelJoin(commands []string, user *User) error {
	if len(commands) != 3 {
		return notEnoughParameters()
	}
	var parameter1 = commands[2]
	var readResponse channelReadResponse
	if len(parameter1) < 1 || parameter1 == PublicChannelName {
		user.CurrentChannelId = ""
		parameter1 = PublicChannelName
	} else {
		jsonResponse, err := json.Marshal(channelGenericDTO{CreatorToken: user.Token, ChannelId: parameter1})
		if err != nil {
			log.Print("handleChannelJoin():", err)
			return errors.New("error joining channel: '" + parameter1 + "'")
		}
		channelResponse, err := apiRequest("POST", newApiRequestOptions(&apiRequestOptions{payload: jsonResponse}), "CHAT_CHANNEL_LIST_URL", nil)
		if err != nil {
			return errors.New("error joining channel: '" + parameter1 + "'")
		}
		json.Unmarshal(channelResponse, &readResponse)
		user.CurrentChannelId = readResponse.Name
	}
	sendToOtherOnChannel(user.Name+" went looking for better content.", user, EventNotification, false, false)
	handleJoin(user)
	sendSystemMessage("Succesfully joined channel '"+parameter1+"'", user, EventNotification)
	return nil
}

func handleChannelList(params []string, user *User) error {
	jsonResponse, err := json.Marshal(channelGenericDTO{CreatorToken: user.Token})
	if err != nil {
		log.Print("handleChannelList():", err)
		return errors.New("error listing channels")
	}
	channelResponse, err := apiRequest("POST", newApiRequestOptions(&apiRequestOptions{payload: jsonResponse}), "CHAT_CHANNEL_LIST_URL", nil)
	if err != nil {
		return errors.New("error listing channels")
	}
	sendSystemMessage(string(channelResponse), user, EventChannelList)
	return nil
}

func handleChannelDefault(params []string, user *User) error {
	if len(user.CurrentChannelId) == 0 {
		return errors.New("you are currently on the 'public' channel which does not need to be set as default")
	}
	jsonResponse, err := json.Marshal(channelGenericDTO{CreatorToken: user.Token, ChannelId: user.CurrentChannelId})
	if err != nil {
		log.Print("handleChannelDefault():", err)
		return errors.New("error setting default channel")
	}
	_, err = apiRequest("PUT", newApiRequestOptions(&apiRequestOptions{payload: jsonResponse}), "CHAT_CHANNEL_DEFAULT_URL", nil)
	if err != nil {
		return errors.New("error setting default channel")
	}
	sendSystemMessage("Successfully set channel: '"+user.CurrentChannelId+"' as your default channel.", user, EventNotification)
	return nil
}