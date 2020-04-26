package main

import (
	"net/http"
	"os"

	"log"

	"github.com/joho/godotenv"
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

func main() {
	initEnvFile()
	initRoutes()
	log.Println("main(): Starting server...")
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Panic(err)
	}
}
