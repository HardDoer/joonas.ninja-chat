package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

const (
	pingWait       = 10 * time.Second
	pongWait       = 60 * time.Second
	maxMessageSize = 512
)

func initEnvFile() {
	var err = godotenv.Load("app.env")
	if err != nil {
		log.Panic("Error loading app.env file. Please create one next to me.")
	}
	log.Print("initEnvFile():", "Loaded envs.")
}

func initRoutes() {
	http.HandleFunc("/api/v1/ws/chat", chatRequest)
	http.HandleFunc("/api/v1/http/chat/login", loginRequest)
	log.Print("initRoutes():", "Routes initialized.")
}

type responseParserFn func(response any) any

/*
Unmarshals a json response
*/
func unmarshalJsonResponse(res []byte) (any, error) {
	var jsonResponse any
	err := json.Unmarshal(res, &jsonResponse)
	if err != nil {
		log.Print("buildJsonResponse():", err)
		return nil, err
	}
	return jsonResponse, err
}

/*
Unmarshals a json response and parses it further with the provided responseParser function
*/
func buildParsedJsonResponse(res []byte, responseParser responseParserFn) []byte {
	jsonResponse, err := unmarshalJsonResponse(res)
	if (err != nil) {
		return nil
	}
	response := responseParser(jsonResponse)
	parsedJsonResponse, err := json.Marshal(response)
	if err != nil {
		log.Print("buildParsedJsonResponse():", err)
		return nil
	}
	return parsedJsonResponse
}

func heartbeat(user *User) {
	log.Print("main():", "Starting heartbeat...")
	defer func() {
		log.Print("heartbeat():", "Stopping heartbeat..")
		user.Connection.Close()
	}()
	for {
		time.Sleep(2 * time.Second)
		if err := user.write(websocket.PingMessage, nil); err != nil {
			log.Print("heartbeat():", err)
			return
		}
	}
}

func main() {
	initEnvFile()
	initRoutes()
	log.Print("main():", "Starting server...")
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Panic(err)
	}
}
