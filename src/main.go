package main

import (
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
	http.HandleFunc("/api/v1/ws/chat", ChatRequest)
	http.HandleFunc("/api/v1/http/chat/login", LoginRequest)
	log.Print("initRoutes():", "Routes initialized.")
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
