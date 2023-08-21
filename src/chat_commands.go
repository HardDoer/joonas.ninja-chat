package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

type helpDTO struct {
	Desc string `json:"description"`
	Name string `json:"name"`
}

type channelDTO struct {
	Name         string `json:"name"`
	CreatorToken string `json:"creatorToken"`
	Private      bool   `json:"private"`
}

type nameChangeDTO struct {
	Username     string `json:"username"`
	CreatorToken string `json:"creatorToken"`
}

type channelInviteDTO struct {
	Name         string `json:"name"`
	CreatorToken string `json:"creatorToken"`
	NewUser      string `json:"newUser"`
}

type channelGenericDTO struct {
	CreatorToken string `json:"creatorToken"`
	ChannelId    string `json:"channelId"`
}

type channelReadResponse struct {
	Name    string `json:"name"`
	Private bool   `json:"private"`
	Admin   string `json:"admin"`
}

type apiRequestOptions struct {
	payload     []byte
	queryString string
	headers     map[string]string
}

func newApiRequestOptions(params *apiRequestOptions) apiRequestOptions {
	a := apiRequestOptions{headers: map[string]string{}}
	if params == nil {
		return a
	}
	if params.payload != nil {
		a.payload = params.payload
	}
	if params.headers != nil {
		a.headers = params.headers
	}
	if a.queryString == "" {
		a.queryString = params.queryString
	}
	return a
}

type responseFn func([]byte) []byte

// handleHelpCommand - dibadaba
func handleHelpCommand(_ []string, user *User) error {
	var response []helpDTO
	response = append(response, helpDTO{Desc: "This command", Name: CommandHelp})
	response = append(response, helpDTO{Desc: "What channel are you on.", Name: CommandWhereAmI})
	response = append(response, helpDTO{Desc: "Logged in users on this channel", Name: CommandWho})
	response = append(response, helpDTO{Desc: "For channel operations. Available parameters are 'invite <channelName> <email>', 'create <channelName>.', 'default' that sets the current channel as your default, 'join <channelName>' and 'list'", Name: CommandChannel})
	response = append(response, helpDTO{Desc: "Change your name. Nickname is only persistent if you are registered and logged in. Parameters: <newName>'", Name: CommandNameChange})
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return err
	}
	sendSystemMessage(string(jsonResponse), user, EventHelp)
	return nil
}

func handleWhereCommand(_ []string, user *User) error {
	if len(user.Token) > 0 {
		var currentChannelId string
		if user.CurrentChannelId == "" {
			currentChannelId = PublicChannelName
		} else {
			currentChannelId = user.CurrentChannelId
		}
		sendSystemMessage("You are currently on channel '"+currentChannelId+"'", user, EventNotification)
	} else {
		return replyMustBeLoggedIn()
	}
	return nil
}

func changeNameRequest(user *User, method string, env string, body string) error {
	jsonResponse, _ := json.Marshal(nameChangeDTO{Username: body, CreatorToken: user.Token})
	_, err := apiRequest(method, newApiRequestOptions(&apiRequestOptions{payload: jsonResponse}), env, func(response []byte) []byte {
		originalName := user.Name
		user.Name = body
		Users.Store(user, user)
		sendSystemMessage(body, user, EventNameChange)
		sendToOtherOnChannel(originalName+" is now called "+body, user, EventNotification, false, false)
		return nil
	})
	return err
}

func handleNameChangeCommand(splitBody []string, u *User) error {
	if len(splitBody) < 2 {
		return nil
	}
	var body = splitBody[1]
	var err error

	if len(body) <= 64 && len(body) >= 1 {
		body = strings.ReplaceAll(body, " ", "")
		if body == "" {
			return errors.New("no empty names")
		}
		key, _ := Users.Load(u)
		user := key.(*User)
		log.Println("handleNameChangeCommand(): User " + user.Name + " is changing name.")
		if user.Name == body {
			return errors.New("you already have that nickname")
		}
		if len(user.Token) > 0 {
			if err := changeNameRequest(user, "PUT", "CHAT_CHANGE_NICKNAME", body); err != nil {
				return errors.New("names must be unique")
			}

		} else {
			if err := changeNameRequest(user, "POST", "CHAT_CHECK_NICKNAME", body); err != nil {
				return errors.New("name reserved by registered user. Register to reserve nicknames")
			}
		}
	} else {
		return errors.New("that name is too long ")
	}
	return err
}

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
	sendToOtherOnChannel(user.Name+" went looking for better content.", user, EventNotification, false, false)
	if len(parameter1) < 1 || parameter1 == PublicChannelName {
		user.CurrentChannelId = ""
		parameter1 = PublicChannelName
	} else {
		jsonResponse, err := json.Marshal(channelGenericDTO{CreatorToken: user.Token, ChannelId: parameter1})
		if err != nil {
			log.Print("handleChannelCommand():", err)
			return errors.New("error joining channel: '" + parameter1 + "'")
		}
		channelResponse, err := apiRequest("POST", newApiRequestOptions(&apiRequestOptions{payload: jsonResponse}), "CHAT_CHANNEL_LIST_URL", nil)
		if err != nil {
			return errors.New("error joining channel: '" + parameter1 + "'")
		}
		json.Unmarshal(channelResponse, &readResponse)
		user.CurrentChannelId = readResponse.Name
	}
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
	client := &http.Client{}
	jsonResponse, _ := json.Marshal(channelGenericDTO{CreatorToken: user.Token, ChannelId: user.CurrentChannelId})
	req, _ := http.NewRequest("PUT", os.Getenv("CHAT_CHANNEL_DEFAULT_URL"), bytes.NewBuffer(jsonResponse))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", `Basic `+
		base64.StdEncoding.EncodeToString([]byte(os.Getenv("APP_ID")+":"+os.Getenv("API_KEY"))))
	channelResponse, err := client.Do(req)
	if err != nil {
		log.Print("handleChannelCommand():", err)
		// Palauta joku virhe
	}
	if channelResponse != nil && channelResponse.Status != "200 OK" {
		log.Print("handleChannelCommand():", "Error response "+channelResponse.Status)
	}
	sendSystemMessage("Successfully set channel: '"+user.CurrentChannelId+"' as your default channel.", user, EventNotification)
	defer channelResponse.Body.Close()
	return nil
}

// handleChannelCommand - dibadaba
func handleChannelCommand(commands []string, user *User) error {
	if len(commands) >= 2 {
		var subCommand = commands[1]
		if len(user.Token) > 0 {
			commandFn, ok := getChannelCommand(subCommand)
			if (!ok) {
				return notEnoughParameters()
			}
			return commandFn(commands, user)
		} else {
			return replyMustBeLoggedIn()
		}
	}
	return notEnoughParameters()
}

// HandleUserCommand - sdgsdfg
func HandleUserCommand(commands []string, connection *websocket.Conn) {

}

// handleWhoCommand - who is present in the current channel
func handleWhoCommand(_ []string, user *User) error {
	var whoIsHere []string
	Users.Range(func(key, value interface{}) bool {
		v := value.(*User)
		if user.CurrentChannelId == v.CurrentChannelId {
			whoIsHere = append(whoIsHere, v.Name)
		}
		return true
	})
	jsonResponse, err := json.Marshal(whoIsHere)
	if err != nil {
		log.Printf("handleWhoCommand(): ")
		log.Println(err)
		return err
	}
	sendSystemMessage(string(jsonResponse), user, EventWho)
	return nil
}
