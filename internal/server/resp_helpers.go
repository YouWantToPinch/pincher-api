package server

import (
	"log"
	"net/http"
	"encoding/json"
)

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {

    if err != nil {
		log.Println(err)
	}

	type errorResponse struct {
		Error string `json:"error"`
	}
	log.Printf("ERROR: %s", msg)
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
	return
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    dat, err := json.Marshal(payload)
	if err != nil {
			log.Printf("Error marshalling JSON for response: %s", err)
			w.WriteHeader(500)
			return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
	return
}

func respondWithHTML(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write([]byte(msg)); err != nil {
		log.Print(err)
	}
	return
}

func respondWithText(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	if _, err := w.Write([]byte(msg)); err != nil {
		log.Print(err)
	}
	return
}