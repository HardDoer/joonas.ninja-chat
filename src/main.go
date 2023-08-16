package main

import (
	"log"
	"net/http"
	"os"
	"time"
	
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

func main() {
	initEnvFile()
	initRoutes()
	log.Print("main():", "Starting server on port: " + os.Getenv("PORT"))
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Panic(err)
	}
}
