package services

import (
	"back/services/common"
	"back/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

type GetElevenLabTTS struct {
	common.CustomHandler
}

func (h *GetElevenLabTTS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Start getElevenLabSpeech handler")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Method '%v' not allowed\n", r.Method)
		log.Println("Close handler")
		return
	}

	h.Ctx = r.Context()

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

func (h *GetElevenLabTTS) getSpeech(w http.ResponseWriter, r *http.Request) error {
	log.Println("Start requesting speech")

	var requestBody common.RequestBody
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
	req, err := speechClientHTTP.InitElevenLabRequest(h.Ctx, requestBody.Text, w)
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

	// Retrieve and stream back the audio response
	w.Header().Set("Content-Type", res.Header.Get("Content-Type"))
	w.WriteHeader(res.StatusCode)

	// Directly copy the response body to the HTTP response writer
	_, err = io.Copy(w, res.Body)
	if err != nil {
		return fmt.Errorf("error copying audio stream to response: %w", err)
	}

	log.Println("Response sent")
	return nil
}
