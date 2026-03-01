// Package utils for utils functions
package utils

import (
	"back/services/common"
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

	apiKey = os.Getenv("ELEVEN_LAB_API_KEY")
	if apiKey == "" {
		log.Fatal("ELEVEN_LAB_API_KEY variable is not defined in .env file")
	}

	apiURL = os.Getenv("ELEVEN_LAB_URL")
	if apiURL == "" {
		log.Fatal("ELEVEN_LAB_URL variable is not defined in .env file")
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

func (c *SpeechClientHTTP) InitGoogleRequest(ctx context.Context, text string, w http.ResponseWriter) (*http.Request, error) {
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

func (c *SpeechClientHTTP) InitElevenLabRequest(ctx context.Context, text string, w http.ResponseWriter) (*http.Request, error) {
	requestBody := common.RequestBody{
		Text: text,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error JSON convert : %v\n", err)
		return nil, err
	}

	// voiceId := "21m00Tcm4TlvDq8ikWAM"
	// voiceId := "CwhRBWXzGAHq8TQ4Fs17" // Roger (fr)
	// voiceId := "9BWtsMINqrJLrRacOk9x" // Aria (fr quebec accent)
	voiceId := "EXAVITQu4vr4xnSDxMaL" // Sarah (ok but with bad intonation)

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%v/%v", apiURL, voiceId), bytes.NewBuffer(jsonData))
	req.Header.Set("voice_id", "21m00Tcm4TlvDq8ikWAM")
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("language_code", "FR")
	// req.Header.Set("output_format", apiKey)

	if err != nil {
		http.Error(w, fmt.Sprintf("Request error: %v", err), http.StatusInternalServerError)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	query := req.URL.Query()
	query.Add("key", apiKey)
	req.URL.RawQuery = query.Encode()

	log.Println(req.URL)

	return req, err
}
