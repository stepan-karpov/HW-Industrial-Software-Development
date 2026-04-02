package handlers

import (
	"io"
	"net/http"
)

func Root(greeting string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, greeting)
	}
}
