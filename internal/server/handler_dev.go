package server

import (
	"log"
	"net/http"
)

func(cfg *apiConfig) endpDeleteAllUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if cfg.platform != "dev" {
		respondWithText(w, 403, "403 Forbidden")
	}

	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		log.Print(err)
	}

	respondWithText(w, 200, "Successfully deleted all users.")
}