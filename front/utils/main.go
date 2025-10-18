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
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

type RequestBody struct {
	Text string `json:"text"`
}

var apiURL string

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error when loading .env file : %v", err)
	}

	apiURL = os.Getenv("API_URL")
	if apiURL == "" {
		log.Fatal("API_URL variable is not defined in .env file")
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

func GetSelectedText() string {
	cmd := exec.Command("xclip", "-o", "-selection", "primary")
	output, err := cmd.Output()
	if err != nil {
		log.Println("Error when getting selected text :", err)
		return ""
	}

	text := strings.TrimSpace(string(output))
	if text == "" {
		log.Println("No text selected")
		return ""
	}

	return text
}

func GetSpeech(ctx context.Context, text string) ([]byte, error) {
	requestBody := RequestBody{Text: text}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("Error JSON convert : %v\n", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return nil, err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request error: %v", err)
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		log.Printf("Server error: %s (code %d)", string(body), res.StatusCode)
		return nil, err
	}

	audioData, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return nil, err
	}

	return audioData, nil
}

func SaveContent(audioData []byte) error {
	err := os.WriteFile("output.wav", audioData, 0644)
	if err != nil {
		log.Printf("Error writing file: %v", err)
		return err
	}

	return nil
}
