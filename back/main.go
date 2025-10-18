// Package main is to interract with user.
// This package communicates with back server in order to process text to speech.
// V1
package main

import (
	"back/utils"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type customHandler struct {
	ctx        context.Context
	mu         sync.Mutex
	cancelFunc context.CancelFunc
}
type getTTS struct {
	customHandler
}
type test struct {
	customHandler
}
type RequestBody struct {
	Text string `json:"text"`
}

func main() {
	utils.LoadEnv()
	utils.InitLogger()

	mux := http.NewServeMux()
	mux.Handle("/tts", &getTTS{})
	mux.Handle("/test", &test{})

	log.Println("Server listening on port :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (h *test) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Start Test handler")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Method '%v' not allowed\n", r.Method)
		log.Println("Close handler")
		return
	}

	h.ctx = r.Context()

	select {
	case <-h.ctx.Done():
		log.Println("Request canceled by client")
		log.Println("Close handler")
		return
	case <-time.After(5 * time.Second):

		var requestBody RequestBody
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			log.Println("Error JSON decode", err)
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		text := strings.TrimSpace(string(requestBody.Text))
		if (text) == "" {
			log.Println("Body text is required")
			return
		}

		log.Println("Received text : ", text)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write([]byte("coucou"))

		log.Println("Response sent")

		log.Println("Close handler")
	}
}

func (h *getTTS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Start getTTS handler")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Method '%v' not allowed\n", r.Method)
		log.Println("Close handler")
		return
	}

	h.ctx = r.Context()

	if err := h.getSpeech(w, r); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Println("Request canceled by client")
			return
		}
		if errors.Is(err, context.DeadlineExceeded) {
			log.Println("Request timed out")
			http.Error(w, "Timeout", http.StatusGatewayTimeout)
			return
		}
		log.Printf("Error in getSpeech: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Println("Close handler")
}

func (h *getTTS) getSpeech(w http.ResponseWriter, r *http.Request) error {
	log.Println("Start requesting speech")

	var requestBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		log.Println("Error JSON decode", err)
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return err
	}

	if requestBody.Text == "" {
		errorText := "the 'text' parameter is required"
		http.Error(w, errorText, http.StatusBadRequest)
		return errors.New(errorText)
	}

	log.Printf("New speach request received : '%s'", requestBody.Text)

	speechClientHTTP := utils.GetSpeechClientHTTP()
	req, err := speechClientHTTP.InitRequest(h.ctx, requestBody.Text, w)
	if err != nil {
		return err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return fmt.Errorf("API error : %s", res.Status)
	}

	audioContent, err := getAudioContent(res, w)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "audio/wav") // "audio/mpeg" for MP3
	w.Header().Set("Content-Disposition", `inline; filename="speech.wav"`)
	w.Write(audioContent)

	log.Println("Response sent")

	return nil
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
