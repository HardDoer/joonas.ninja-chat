package main

import (
	"net/http"
	"os"

	"log"
	"time"
	"github.com/joho/godotenv"
	"github.com/gorilla/websocket"
)

func initEnvFile() {
	var err = godotenv.Load("app.env")
	if err != nil {
		log.Panic("Error loading app.env file. Please create one next to me.")
	}
	log.Println("initEnvFile(): Loaded envs.")
}

func initRoutes() {
	http.HandleFunc("/api/v1/ws/chat", ChatRequest)
	log.Println("initRoutes(): Routes initialized.")
}

func heartbeat() {
	for {
		time.Sleep(2 * time.Second)
		log.Println("PING")
		for i := 0; i < len(Users); i ++ {
			if err := Users[i].Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func main() {
	initEnvFile()
	initRoutes()
	log.Println("main(): Starting server...")
	go heartbeat()
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Panic(err)
	}
}
