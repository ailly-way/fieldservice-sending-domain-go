package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	api := &apiClient{key: key, base: baseURL, http: &http.Client{Timeout: 15 * time.Second}}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /dispatch", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var in dispatchRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		result, err := dispatch(r.Context(), api, in)
		if err != nil {
			code := http.StatusBadGateway
			var remote *apiError
			if errors.As(err, &remote) {
				code = http.StatusBadRequest
			} else if result.ZoneID == "" {
				code = http.StatusBadRequest
			}
			http.Error(w, err.Error(), code)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if result.MessageID == "" {
			w.WriteHeader(http.StatusAccepted)
		}
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("encode response: %v", err)
		}
	})
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Fatal(http.ListenAndServe(addr, mux))
}
