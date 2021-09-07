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
	pingWait = 10 * time.Second
	pongWait = 60 * time.Second
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
	log.Print("initRoutes():", "Routes initialized.")
}

func heartbeat() {
	for {
		time.Sleep(2 * time.Second)
		if UserCount > 0 {
			Users.Range(func(key, value interface{}) bool {
				var userValue = value.(*User)
				if err := userValue.write(websocket.PingMessage, nil); err != nil {
					log.Print("heartbeat():", err)
				}
				return true
			})
		}
	}
}

func main() {
	initEnvFile()
	initRoutes()
	log.Print("main():", "Starting server...")
	log.Print("main():", "Starting heartbeat...")
	go heartbeat()
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Panic(err)
	}
}
