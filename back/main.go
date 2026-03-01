// Package main is to interract with user.
// This package communicates with back server in order to process text to speech.
// V1
package main

import (
	"back/services"
	"back/utils"
	"log"
	"net/http"
)

func main() {
	utils.LoadEnv()
	utils.InitLogger()

	mux := http.NewServeMux()
	// mux.Handle("/tts", &services.GetTTS{})
	mux.Handle("/test", &services.Test{})
	mux.Handle("/tts", &services.GetElevenLabTTS{})

	log.Println("Server listening on port :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
