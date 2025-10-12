// Package main is to interract with user.
// This package communicates with back server in order to process text to speech.

// V1

package main

import (
	"io"
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
	abort = make(chan struct{})

	http.HandleFunc("/tts", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		if cmd != nil {
			log.Println("Stopping any ongoing speach")
			cmd.Process.Kill()
			cmd = nil
			close(abort)
			abort = make(chan struct{})
		}
		mu.Unlock()

		text := r.URL.Query().Get("text")
		if text == "" {
			http.Error(w, "The 'text' parameter is required", http.StatusBadRequest)
			return
		}

		log.Printf("New speach request received : '%s'", text)

		cmd = exec.Command("espeak-ng", "-v", "fr", "--stdout", text)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Error creating stdout pipe: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err := cmd.Start(); err != nil {
			log.Printf("Error starting espeak-ng: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "audio/wav")
		w.Header().Set("Transfer-Encoding", "chunked")
		log.Println("Streaming audio started")

		buf := make([]byte, 1024)
		totalBytes := 0

		for {
			select {
			case <-abort:
				log.Println("Speech interrupted by new request")
				cmd.Process.Kill()
				return
			default:
				n, err := stdout.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Error reading audio stream: %v", err)
					} else {
						log.Printf("Audio stream completed. Total bytes sent: %d", totalBytes)
					}
					mu.Lock()
					cmd = nil
					mu.Unlock()
					return
				}

				totalBytes += n
				if _, err := w.Write(buf[:n]); err != nil {
					log.Printf("Error sending audio data: %v", err)
					mu.Lock()
					cmd = nil
					mu.Unlock()
					return
				}

				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
			}
		}
	})

	log.Println("Server listening on port :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
