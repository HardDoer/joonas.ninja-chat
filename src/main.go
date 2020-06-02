package main

import (
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"log"
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

func main() {
	initEnvFile()
	initRoutes()
	log.Print("main():", "Starting server...")
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Panic(err)
	}
}
