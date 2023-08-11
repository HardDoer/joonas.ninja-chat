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

type structConstructorFn func(response any) any

func buildJsonResponse(res []byte, castObject any, responseConstructor structConstructorFn) [] byte {
	err := json.Unmarshal(res, &castObject)
	if err != nil {
		log.Print("buildJsonResponse():", err)
		return nil
	}
	if (responseConstructor != nil) {
		response := responseConstructor(castObject)
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			log.Print("buildJsonResponse():", err)
			return nil
		}
		return jsonResponse
	}
	return nil
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
