package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type server struct {
	maxFileSize int64
}

func main() {
	maxFileSize := int64(50 * 1024 * 1024 * 1024)
	if value := os.Getenv("MAX_FILE_SIZE_BYTES"); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			maxFileSize = parsed
		}
	}

	s := &server{maxFileSize: maxFileSize}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/config", s.config)

	address := ":" + getenv("API_PORT", "8080")
	log.Printf("filetransfer API listening on %s", address)
	if err := http.ListenAndServe(address, logging(mux)); err != nil {
		log.Fatal(err)
	}
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "filetransfer-api",
	})
}

func (s *server) config(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"maxFileSizeBytes": s.maxFileSize,
		"maxFileSizeGB":    50,
		"uploadProtocol":   "tus",
		"locale":           "fa-IR",
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started))
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
