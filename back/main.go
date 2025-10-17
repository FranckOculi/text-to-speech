// Package main is to interract with user.
// This package communicates with back server in order to process text to speech.
// V1
package main

import (
	"back/utils"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"sync"
)

var (
	mu    sync.Mutex
	cmd   *exec.Cmd
	abort chan struct{}
)

func main() {
	utils.LoadEnv()
	abort = make(chan struct{})
	http.HandleFunc("/tts", func(w http.ResponseWriter, r *http.Request) {
		abortCurrentRequest()
		getSpeech(w, r)
	})

	log.Println("Server listening on port :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func getSpeech(w http.ResponseWriter, r *http.Request) {
	log.Println("Start requesting speech")

	text := r.URL.Query().Get("text")
	if text == "" {
		http.Error(w, "The 'text' parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("New speach request received : '%s'", text)

	speechClientHTTP := utils.GetSpeechClientHTTP()
	req, err := speechClientHTTP.InitRequest(text, w)

	if err != nil {
		return
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("API error: %s (code %d)", res.Status, res.StatusCode), res.StatusCode)
		return
	}

	defer res.Body.Close()

	if res.StatusCode >= 400 {
		log.Fatalf("Erreur de l'API : %s", res.Status)
	}

	audioContent, err := getAudioContent(res, w)
	if err != nil {
		return
	}

	w.Header().Set("Content-Type", "audio/wav") // "audio/mpeg" for MP3
	w.Header().Set("Content-Disposition", `inline; filename="speech.wav"`)
	w.Write(audioContent)
	log.Println("Response sent")
}

func getAudioContent(res *http.Response, w http.ResponseWriter) ([]byte, error) {
	var result map[string]any
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		http.Error(w, fmt.Sprintf("Decode error: %v", err), http.StatusInternalServerError)
		return nil, err
	}

	audioContent, ok := result["audioContent"].(string)
	if !ok {
		http.Error(w, "AudioContent not found in response", http.StatusInternalServerError)
		return nil, errors.New("AudioContent not found in response")
	}

	audioBytes, err := base64.StdEncoding.DecodeString(audioContent)
	if err != nil {
		http.Error(w, fmt.Sprintf("Base64 decode error: %v", err), http.StatusInternalServerError)
		return nil, err
	}

	return audioBytes, nil
}

func abortCurrentRequest() {
	mu.Lock()
	if cmd != nil {
		log.Println("Stopping ongoing speach")
		cmd.Process.Kill()
		cmd = nil
		close(abort)
		abort = make(chan struct{})
	}
	mu.Unlock()
}
