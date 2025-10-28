package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {

	if err != nil {
		slog.Error(err.Error())
	}

	type errorResponse struct {
		Error string `json:"error"`
	}
	slog.Error(msg)
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Could not marshal JSON for response: " + err.Error())
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithHTML(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write([]byte(msg)); err != nil {
		slog.Error(err.Error())
	}
}

func respondWithText(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write([]byte(msg)); err != nil {
		slog.Error(err.Error())
	}
}
