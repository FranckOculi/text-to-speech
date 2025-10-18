// Package utils for utils functions
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

var (
	apiKey, apiURL string
	clientHTTPMu   sync.Mutex
	clientHTTP     *SpeechClientHTTP
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error when loading .env file : %v", err)
	}

	apiKey = os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY variable is not defined in .env file")
	}

	apiURL = os.Getenv("GOOGLE_API_URL")
	if apiURL == "" {
		log.Fatal("GOOGLE_API_URL variable is not defined in .env file")
	}
}

func InitLogger() {
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file : %v\n", err)
	}

	mw := io.MultiWriter(os.Stdout, logFile)

	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type SpeechClientHTTP struct {
}

func GetSpeechClientHTTP() *SpeechClientHTTP {
	if clientHTTP == nil {
		log.Println("Creating single instance of client http")
		clientHTTPMu.Lock()
		clientHTTP = &SpeechClientHTTP{}
		clientHTTPMu.Unlock()
	} else {
		log.Println("Single instance of client http already created")
	}

	return clientHTTP
}

func (c *SpeechClientHTTP) InitRequest(ctx context.Context, text string, w http.ResponseWriter) (*http.Request, error) {
	requestBody := map[string]any{
		"input": map[string]string{
			"text": text,
		},
		"voice": map[string]string{
			"languageCode": "fr-FR",
			"ssmlGender":   "MALE",
			"name":         "fr-FR-Chirp3-HD-Algenib",
		},
		"audioConfig": map[string]string{
			// "audioEncoding": "MP3",
			"audioEncoding": "LINEAR16", // Format WAV (PCM 16 bits)
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error JSON convert : %v\n", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", apiKey)

	if err != nil {
		http.Error(w, fmt.Sprintf("Request error: %v", err), http.StatusInternalServerError)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	query := req.URL.Query()
	query.Add("key", apiKey)
	req.URL.RawQuery = query.Encode()

	return req, err
}
