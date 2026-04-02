package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// LogRequest is the JSON body for POST /log.
type LogRequest struct {
	Message string `json:"message"`
}

func LogPost(logPath string, mu *sync.Mutex, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		msg := strings.TrimSpace(req.Message)
		if msg == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
			return
		}

		line := fmt.Sprintf("%s %s\n", time.Now().Format(time.RFC3339), msg)
		mu.Lock()
		err := appendToFile(logPath, line)
		mu.Unlock()
		if err != nil {
			if logger != nil {
				logger.Printf("failed to write log: %v", err)
			}
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to write log"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "written"})
	}
}

func appendToFile(path, s string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(s)
	return err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
