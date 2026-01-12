package api

import (
	"net/http"
)

func (cfg *APIConfig) handleReadiness(w http.ResponseWriter, r *http.Request) {
	respondWithText(w, http.StatusOK, "OK")
}
