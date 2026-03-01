package services

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	authorization "back/middlewares"
	"back/services/common"
)

type Test struct {
	common.CustomHandler
}

// Provide a text response to test api packages beahavior before to implements them
func (h *Test) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Start Test handler")

	authorization.InitAuthentication()
	value, res := authorization.VerifyToken("coucou")

	log.Printf("token : %v - %v", value, res)

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Method '%v' not allowed\n", r.Method)
		log.Println("Close handler")
		return
	}

	h.Ctx = r.Context()

	select {
	case <-h.Ctx.Done():
		log.Println("Request canceled by client")
		log.Println("Close handler")
		return
	case <-time.After(5 * time.Second):

		var requestBody common.RequestBody
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
